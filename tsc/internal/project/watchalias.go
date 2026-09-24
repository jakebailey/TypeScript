package project

import (
	"fmt"
	"slices"
	"strings"

	"github.com/microsoft/TypeScript/tsc/internal/collections"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/ls/lsconv"
	"github.com/microsoft/TypeScript/tsc/internal/lsp/lsproto"
	"github.com/microsoft/TypeScript/tsc/internal/project/logging"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
	"github.com/microsoft/TypeScript/tsc/internal/watchalias"
)

// Build only for watch notifications, not for editor-only snapshot changes.
func (s *Snapshot) getNativeWatchIndex() (*watchalias.Index, error) {
	if !s.host.options.WatchEnabled || s.fileSystemOverride || !watchalias.Enabled(s.host.fs) {
		return nil, nil
	}
	s.nativeWatchIndexOnce.Do(func() {
		index := watchalias.New(s.host.fs)
		directories := make(map[string]string)
		var resolveDirectory func(string) string
		resolveDirectory = func(name string) string {
			if resolved := directories[name]; resolved != "" {
				return resolved
			}
			resolved := vfs.RealpathWithParent(s.host.fs, name, resolveDirectory)
			directories[name] = resolved
			return resolved
		}
		add := func(name string) {
			if s.nativeWatchIndexError != nil || name == "" || tspath.IsDynamicFileName(name) {
				return
			}
			name = tspath.GetNormalizedAbsolutePath(name, s.host.options.CurrentDirectory)
			if err := index.Add(name); err != nil {
				s.nativeWatchIndexError = fmt.Errorf("indexing project watch path %q: %w", name, err)
				return
			}
			// Watch clients can rebase events to symlinked roots. Observe
			// directories explicitly rather than infer them from file links.
			for directory := tspath.GetDirectoryPath(name); directory != ""; {
				registration := watchalias.Registration{
					Name: directory, Realpath: resolveDirectory(directory), Directory: true,
				}
				if index.Covers(registration) {
					break
				}
				if err := index.Register(registration); err != nil {
					s.nativeWatchIndexError = fmt.Errorf("indexing project watch directory %q: %w", directory, err)
					return
				}
				parent := tspath.GetDirectoryPath(directory)
				if parent == directory {
					break
				}
				directory = parent
			}
		}
		for _, file := range s.fs.cacheFiles {
			add(file.FileName())
			if file.realpathName != "" && s.nativeWatchIndexError == nil {
				// Retain physical descendants for directory deletion notifications.
				// Logical correspondence remains in SnapshotFS's reverse alias map.
				if err := index.Register(watchalias.Registration{
					Name: file.realpathName, Realpath: file.realpathName, Dependency: true,
				}); err != nil {
					s.nativeWatchIndexError = fmt.Errorf("indexing project watch path %q: %w", file.realpathName, err)
				}
			}
		}
		for _, overlay := range s.overlays() {
			add(overlay.FileName())
		}
		for _, config := range s.ConfigFileRegistry.configs {
			add(config.fileName)
			if config.commandLine != nil {
				for directory := range config.commandLine.WildcardDirectories() {
					add(directory)
				}
			}
		}
		for _, project := range s.ProjectCollection.Projects() {
			if project.host != nil {
				for _, names := range []*collections.SyncMap[tspath.Path, string]{
					project.host.sourceFS.seenFiles,
					project.host.sourceFS.missingDirectories,
				} {
					if names != nil {
						names.Range(func(_ tspath.Path, name string) bool {
							add(name)
							return s.nativeWatchIndexError == nil
						})
					}
				}
			}
			if project.contentMapperWatch != nil {
				for _, name := range project.contentMapperWatch.input {
					add(name)
				}
			}
		}
		if s.autoImportsWatch != nil {
			for _, name := range s.autoImportsWatch.input {
				add(name)
			}
		}
		if s.nativeWatchIndexError == nil {
			s.nativeWatchIndex = index
		}
	})
	return s.nativeWatchIndex, s.nativeWatchIndexError
}

func (s *Snapshot) expandNativeWatchNotifications(changes []FileChange) ([]FileChange, error) {
	if !slices.ContainsFunc(changes, func(change FileChange) bool { return change.Kind.IsWatchKind() }) {
		return changes, nil
	}
	index, err := s.getNativeWatchIndex()
	if index == nil && err == nil {
		return changes, err
	}
	result := make([]FileChange, 0, len(changes))
	for _, change := range changes {
		if !change.Kind.IsWatchKind() {
			result = append(result, change)
			continue
		}
		if err != nil {
			for _, overlay := range s.overlays() {
				result = append(result, FileChange{
					Kind: FileChangeKindWatchChange,
					URI:  lsconv.FileNameToDocumentURI(overlay.FileName()),
				})
			}
			result = append(result, change)
			continue
		}
		for _, name := range index.Expand(change.URI.FileName()) {
			result = append(result, FileChange{Kind: change.Kind, URI: lsconv.FileNameToDocumentURI(name)})
		}
	}
	return result, err
}

func (s *Snapshot) expandNativeWatchSummary(change FileChangeSummary, logger *logging.LogTree) FileChangeSummary {
	if !change.hasWatchChanges || change.InvalidateAll || change.Created.Len()+change.Changed.Len()+change.Deleted.Len() == 0 {
		return change
	}

	index, err := s.getNativeWatchIndex()
	if err != nil {
		logger.Logf("Native watch names unavailable; invalidating cached project state: %v", err)
		change.InvalidateAll = true
		return change
	}
	if index == nil {
		return change
	}
	expand := func(uris collections.Set[lsproto.DocumentUri], kind fswatch.EventKind) collections.Set[lsproto.DocumentUri] {
		if uris.Len() == 0 {
			return uris
		}
		events := make(map[string]fswatch.EventKind, uris.Len())
		for uri := range uris.Keys() {
			events[uri.FileName()] = kind
		}
		var result collections.Set[lsproto.DocumentUri]
		for name := range index.MatchExpanded(events).Changes {
			result.Add(lsconv.FileNameToDocumentURI(name))
			if !strings.Contains(name, "/node_modules/") {
				change.IncludesWatchChangeOutsideNodeModules = true
			}
		}
		return result
	}
	change.Created = expand(change.Created, fswatch.EventUpdate)
	change.Changed = expand(change.Changed, fswatch.EventUpdate)
	change.Deleted = expand(change.Deleted, fswatch.EventDelete)
	return change
}

func (s *Snapshot) watchChangesOverlapProjectState(change FileChangeSummary) bool {
	for _, events := range []collections.Set[lsproto.DocumentUri]{change.Changed, change.Deleted} {
		for uri := range events.Keys() {
			path := s.host.toPath(uri.FileName())
			base := tspath.GetBaseFileName(string(path))
			if base == "tsconfig.json" || base == "jsconfig.json" || base == s.ConfigFileRegistry.customConfigFileName {
				return true
			}
			if s.ConfigFileRegistry.isTracked(path) || s.hasOverlayWithin(path) {
				return true
			}
			for _, project := range s.ProjectCollection.Projects() {
				if project.host != nil && project.host.sourceFS.SeenFileOrMissingParentDirectory(path) {
					return true
				}
			}
		}
	}
	return false
}
