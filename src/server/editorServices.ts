import * as ts from "./_namespaces/ts";

export const maxProgramSizeForNonTsFiles = 20 * 1024 * 1024;
/*@internal*/
export const maxFileSize = 4 * 1024 * 1024;

export const ProjectsUpdatedInBackgroundEvent = "projectsUpdatedInBackground";
export const ProjectLoadingStartEvent = "projectLoadingStart";
export const ProjectLoadingFinishEvent = "projectLoadingFinish";
export const LargeFileReferencedEvent = "largeFileReferenced";
export const ConfigFileDiagEvent = "configFileDiag";
export const ProjectLanguageServiceStateEvent = "projectLanguageServiceState";
export const ProjectInfoTelemetryEvent = "projectInfo";
export const OpenFileInfoTelemetryEvent = "openFileInfo";
const ensureProjectForOpenFileSchedule = "*ensureProjectForOpenFiles*";

export interface ProjectsUpdatedInBackgroundEvent {
    eventName: typeof ProjectsUpdatedInBackgroundEvent;
    data: { openFiles: string[]; };
}

export interface ProjectLoadingStartEvent {
    eventName: typeof ProjectLoadingStartEvent;
    data: { project: ts.server.Project; reason: string; };
}

export interface ProjectLoadingFinishEvent {
    eventName: typeof ProjectLoadingFinishEvent;
    data: { project: ts.server.Project; };
}

export interface LargeFileReferencedEvent {
    eventName: typeof LargeFileReferencedEvent;
    data: { file: string; fileSize: number; maxFileSize: number; };
}

export interface ConfigFileDiagEvent {
    eventName: typeof ConfigFileDiagEvent;
    data: { triggerFile: string, configFileName: string, diagnostics: readonly ts.Diagnostic[] };
}

export interface ProjectLanguageServiceStateEvent {
    eventName: typeof ProjectLanguageServiceStateEvent;
    data: { project: ts.server.Project, languageServiceEnabled: boolean };
}

/** This will be converted to the payload of a protocol.TelemetryEvent in session.defaultEventHandler. */
export interface ProjectInfoTelemetryEvent {
    readonly eventName: typeof ProjectInfoTelemetryEvent;
    readonly data: ProjectInfoTelemetryEventData;
}

/* __GDPR__
    "projectInfo" : {
        "${include}": ["${TypeScriptCommonProperties}"],
        "projectId": { "classification": "EndUserPseudonymizedInformation", "purpose": "FeatureInsight", "endpoint": "ProjectId" },
        "fileStats": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "compilerOptions": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "extends": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "files": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "include": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "exclude": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "compileOnSave": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "typeAcquisition": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "configFileName": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "projectType": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "languageServiceEnabled": { "classification": "SystemMetaData", "purpose": "FeatureInsight" },
        "version": { "classification": "SystemMetaData", "purpose": "FeatureInsight" }
    }
 */
export interface ProjectInfoTelemetryEventData {
    /** Cryptographically secure hash of project file location. */
    readonly projectId: string;
    /** Count of file extensions seen in the project. */
    readonly fileStats: FileStats;
    /**
     * Any compiler options that might contain paths will be taken out.
     * Enum compiler options will be converted to strings.
     */
    readonly compilerOptions: ts.CompilerOptions;
    // "extends", "files", "include", or "exclude" will be undefined if an external config is used.
    // Otherwise, we will use "true" if the property is present and "false" if it is missing.
    readonly extends: boolean | undefined;
    readonly files: boolean | undefined;
    readonly include: boolean | undefined;
    readonly exclude: boolean | undefined;
    readonly compileOnSave: boolean;
    readonly typeAcquisition: ProjectInfoTypeAcquisitionData;

    readonly configFileName: "tsconfig.json" | "jsconfig.json" | "other";
    readonly projectType: "external" | "configured";
    readonly languageServiceEnabled: boolean;
    /** TypeScript version used by the server. */
    readonly version: string;
}

/**
 * Info that we may send about a file that was just opened.
 * Info about a file will only be sent once per session, even if the file changes in ways that might affect the info.
 * Currently this is only sent for '.js' files.
 */
export interface OpenFileInfoTelemetryEvent {
    readonly eventName: typeof OpenFileInfoTelemetryEvent;
    readonly data: OpenFileInfoTelemetryEventData;
}

export interface OpenFileInfoTelemetryEventData {
    readonly info: OpenFileInfo;
}

export interface ProjectInfoTypeAcquisitionData {
    readonly enable: boolean | undefined;
    // Actual values of include/exclude entries are scrubbed.
    readonly include: boolean;
    readonly exclude: boolean;
}

export interface FileStats {
    readonly js: number;
    readonly jsSize?: number;

    readonly jsx: number;
    readonly jsxSize?: number;

    readonly ts: number;
    readonly tsSize?: number;

    readonly tsx: number;
    readonly tsxSize?: number;

    readonly dts: number;
    readonly dtsSize?: number;

    readonly deferred: number;
    readonly deferredSize?: number;
}

export interface OpenFileInfo {
    readonly checkJs: boolean;
}

export type ProjectServiceEvent =
    LargeFileReferencedEvent
    | ProjectsUpdatedInBackgroundEvent
    | ProjectLoadingStartEvent
    | ProjectLoadingFinishEvent
    | ConfigFileDiagEvent
    | ProjectLanguageServiceStateEvent
    | ProjectInfoTelemetryEvent
    | OpenFileInfoTelemetryEvent;

export type ProjectServiceEventHandler = (event: ProjectServiceEvent) => void;

/*@internal*/
export type PerformanceEventHandler = (event: ts.PerformanceEvent) => void;

export interface SafeList {
    [name: string]: { match: RegExp, exclude?: (string | number)[][], types?: string[] };
}

function prepareConvertersForEnumLikeCompilerOptions(commandLineOptions: ts.CommandLineOption[]): ts.ESMap<string, ts.ESMap<string, number>> {
    const map = new ts.Map<string, ts.ESMap<string, number>>();
    for (const option of commandLineOptions) {
        if (typeof option.type === "object") {
            const optionMap = option.type as ts.ESMap<string, number>;
            // verify that map contains only numbers
            optionMap.forEach(value => {
                ts.Debug.assert(typeof value === "number");
            });
            map.set(option.name, optionMap);
        }
    }
    return map;
}

const compilerOptionConverters = prepareConvertersForEnumLikeCompilerOptions(ts.optionDeclarations);
const watchOptionsConverters = prepareConvertersForEnumLikeCompilerOptions(ts.optionsForWatch);
const indentStyle = new ts.Map(ts.getEntries({
    none: ts.IndentStyle.None,
    block: ts.IndentStyle.Block,
    smart: ts.IndentStyle.Smart
}));

export interface TypesMapFile {
    typesMap: SafeList;
    simpleMap: { [libName: string]: string };
}

/**
 * How to understand this block:
 *  * The 'match' property is a regexp that matches a filename.
 *  * If 'match' is successful, then:
 *     * All files from 'exclude' are removed from the project. See below.
 *     * All 'types' are included in ATA
 *  * What the heck is 'exclude' ?
 *     * An array of an array of strings and numbers
 *     * Each array is:
 *       * An array of strings and numbers
 *       * The strings are literals
 *       * The numbers refer to capture group indices from the 'match' regexp
 *          * Remember that '1' is the first group
 *       * These are concatenated together to form a new regexp
 *       * Filenames matching these regexps are excluded from the project
 * This default value is tested in tsserverProjectSystem.ts; add tests there
 *   if you are changing this so that you can be sure your regexp works!
 */
const defaultTypeSafeList: SafeList = {
    "jquery": {
        // jquery files can have names like "jquery-1.10.2.min.js" (or "jquery.intellisense.js")
        match: /jquery(-[\d\.]+)?(\.intellisense)?(\.min)?\.js$/i,
        types: ["jquery"]
    },
    "WinJS": {
        // e.g. c:/temp/UWApp1/lib/winjs-4.0.1/js/base.js
        match: /^(.*\/winjs-[.\d]+)\/js\/base\.js$/i,        // If the winjs/base.js file is found..
        exclude: [["^", 1, "/.*"]],                // ..then exclude all files under the winjs folder
        types: ["winjs"]                           // And fetch the @types package for WinJS
    },
    "Kendo": {
        // e.g. /Kendo3/wwwroot/lib/kendo/kendo.all.min.js
        match: /^(.*\/kendo(-ui)?)\/kendo\.all(\.min)?\.js$/i,
        exclude: [["^", 1, "/.*"]],
        types: ["kendo-ui"]
    },
    "Office Nuget": {
        // e.g. /scripts/Office/1/excel-15.debug.js
        match: /^(.*\/office\/1)\/excel-\d+\.debug\.js$/i, // Office NuGet package is installed under a "1/office" folder
        exclude: [["^", 1, "/.*"]],                     // Exclude that whole folder if the file indicated above is found in it
        types: ["office"]                               // @types package to fetch instead
    },
    "References": {
        match: /^(.*\/_references\.js)$/i,
        exclude: [["^", 1, "$"]]
    }
};

export function convertFormatOptions(protocolOptions: ts.server.protocol.FormatCodeSettings): ts.FormatCodeSettings {
    if (ts.isString(protocolOptions.indentStyle)) {
        protocolOptions.indentStyle = indentStyle.get(protocolOptions.indentStyle.toLowerCase());
        ts.Debug.assert(protocolOptions.indentStyle !== undefined);
    }
    return protocolOptions as any;
}

export function convertCompilerOptions(protocolOptions: ts.server.protocol.ExternalProjectCompilerOptions): ts.CompilerOptions & ts.server.protocol.CompileOnSaveMixin {
    compilerOptionConverters.forEach((mappedValues, id) => {
        const propertyValue = protocolOptions[id];
        if (ts.isString(propertyValue)) {
            protocolOptions[id] = mappedValues.get(propertyValue.toLowerCase());
        }
    });
    return protocolOptions as any;
}

export function convertWatchOptions(protocolOptions: ts.server.protocol.ExternalProjectCompilerOptions, currentDirectory?: string): WatchOptionsAndErrors | undefined {
    let watchOptions: ts.WatchOptions | undefined;
    let errors: ts.Diagnostic[] | undefined;
    ts.optionsForWatch.forEach(option => {
        const propertyValue = protocolOptions[option.name];
        if (propertyValue === undefined) return;
        const mappedValues = watchOptionsConverters.get(option.name);
        (watchOptions || (watchOptions = {}))[option.name] = mappedValues ?
            ts.isString(propertyValue) ? mappedValues.get(propertyValue.toLowerCase()) : propertyValue :
            ts.convertJsonOption(option, propertyValue, currentDirectory || "", errors || (errors = []));
    });
    return watchOptions && { watchOptions, errors };
}

export function convertTypeAcquisition(protocolOptions: ts.server.protocol.InferredProjectCompilerOptions): ts.TypeAcquisition | undefined {
    let result: ts.TypeAcquisition | undefined;
    ts.typeAcquisitionDeclarations.forEach((option) => {
        const propertyValue = protocolOptions[option.name];
        if (propertyValue === undefined) return;
        (result || (result = {}))[option.name] = propertyValue;
    });
    return result;
}

export function tryConvertScriptKindName(scriptKindName: ts.server.protocol.ScriptKindName | ts.ScriptKind): ts.ScriptKind {
    return ts.isString(scriptKindName) ? convertScriptKindName(scriptKindName) : scriptKindName;
}

export function convertScriptKindName(scriptKindName: ts.server.protocol.ScriptKindName) {
    switch (scriptKindName) {
        case "JS":
            return ts.ScriptKind.JS;
        case "JSX":
            return ts.ScriptKind.JSX;
        case "TS":
            return ts.ScriptKind.TS;
        case "TSX":
            return ts.ScriptKind.TSX;
        default:
            return ts.ScriptKind.Unknown;
    }
}

/*@internal*/
export function convertUserPreferences(preferences: ts.server.protocol.UserPreferences): ts.UserPreferences {
    const { lazyConfiguredProjectsFromExternalProject, ...userPreferences } = preferences;
    return userPreferences;
}

export interface HostConfiguration {
    formatCodeOptions: ts.FormatCodeSettings;
    preferences: ts.server.protocol.UserPreferences;
    hostInfo: string;
    extraFileExtensions?: ts.FileExtensionInfo[];
    watchOptions?: ts.WatchOptions;
}

export interface OpenConfiguredProjectResult {
    configFileName?: ts.server.NormalizedPath;
    configFileErrors?: readonly ts.Diagnostic[];
}

interface AssignProjectResult extends OpenConfiguredProjectResult {
    retainProjects: readonly ts.server.ConfiguredProject[] | ts.server.ConfiguredProject | undefined;
}

interface FilePropertyReader<T> {
    getFileName(f: T): string;
    getScriptKind(f: T, extraFileExtensions?: ts.FileExtensionInfo[]): ts.ScriptKind;
    hasMixedContent(f: T, extraFileExtensions: ts.FileExtensionInfo[] | undefined): boolean;
}

const fileNamePropertyReader: FilePropertyReader<string> = {
    getFileName: x => x,
    getScriptKind: (fileName, extraFileExtensions) => {
        let result: ts.ScriptKind | undefined;
        if (extraFileExtensions) {
            const fileExtension = ts.getAnyExtensionFromPath(fileName);
            if (fileExtension) {
                ts.some(extraFileExtensions, info => {
                    if (info.extension === fileExtension) {
                        result = info.scriptKind;
                        return true;
                    }
                    return false;
                });
            }
        }
        return result!; // TODO: GH#18217
    },
    hasMixedContent: (fileName, extraFileExtensions) => ts.some(extraFileExtensions, ext => ext.isMixedContent && ts.fileExtensionIs(fileName, ext.extension)),
};

const externalFilePropertyReader: FilePropertyReader<ts.server.protocol.ExternalFile> = {
    getFileName: x => x.fileName,
    getScriptKind: x => tryConvertScriptKindName(x.scriptKind!), // TODO: GH#18217
    hasMixedContent: x => !!x.hasMixedContent,
};

function findProjectByName<T extends ts.server.Project>(projectName: string, projects: T[]): T | undefined {
    for (const proj of projects) {
        if (proj.getProjectName() === projectName) {
            return proj;
        }
    }
}

const noopConfigFileWatcher: ts.FileWatcher = { close: ts.noop };

/*@internal*/
interface ConfigFileExistenceInfo {
    /**
     * Cached value of existence of config file
     * It is true if there is configured project open for this file.
     * It can be either true or false if this is the config file that is being watched by inferred project
     *   to decide when to update the structure so that it knows about updating the project for its files
     *   (config file may include the inferred project files after the change and hence may be wont need to be in inferred project)
     */
    exists: boolean;
    /**
     * openFilesImpactedByConfigFiles is a map of open files that would be impacted by this config file
     *   because these are the paths being looked up for their default configured project location
     * The value in the map is true if the open file is root of the inferred project
     * It is false when the open file that would still be impacted by existence of
     *   this config file but it is not the root of inferred project
     */
    openFilesImpactedByConfigFile?: ts.ESMap<ts.Path, boolean>;
    /**
     * The file watcher watching the config file because there is open script info that is root of
     * inferred project and will be impacted by change in the status of the config file
     * or
     * Configured project for this config file is open
     * or
     * Configured project references this config file
     */
    watcher?: ts.FileWatcher;
    /**
     * Cached parsed command line and other related information like watched directories etc
     */
    config?: ParsedConfig;
}

export interface ProjectServiceOptions {
    host: ts.server.ServerHost;
    logger: ts.server.Logger;
    cancellationToken: ts.HostCancellationToken;
    useSingleInferredProject: boolean;
    useInferredProjectPerProjectRoot: boolean;
    typingsInstaller: ts.server.ITypingsInstaller;
    eventHandler?: ProjectServiceEventHandler;
    suppressDiagnosticEvents?: boolean;
    throttleWaitMilliseconds?: number;
    globalPlugins?: readonly string[];
    pluginProbeLocations?: readonly string[];
    allowLocalPluginLoads?: boolean;
    typesMapLocation?: string;
    /** @deprecated use serverMode instead */
    syntaxOnly?: boolean;
    serverMode?: ts.LanguageServiceMode;
    session: ts.server.Session<unknown> | undefined;
}

interface OriginalFileInfo { fileName: ts.server.NormalizedPath; path: ts.Path; }
interface AncestorConfigFileInfo {
    /** config file name */
    fileName: string;
    /** path of open file so we can look at correct root */
    path: ts.Path;
    configFileInfo: true;
}
type OpenScriptInfoOrClosedFileInfo = ts.server.ScriptInfo | OriginalFileInfo;
type OpenScriptInfoOrClosedOrConfigFileInfo = OpenScriptInfoOrClosedFileInfo | AncestorConfigFileInfo;

function isOpenScriptInfo(infoOrFileNameOrConfig: OpenScriptInfoOrClosedOrConfigFileInfo): infoOrFileNameOrConfig is ts.server.ScriptInfo {
    return !!(infoOrFileNameOrConfig as ts.server.ScriptInfo).containingProjects;
}

function isAncestorConfigFileInfo(infoOrFileNameOrConfig: OpenScriptInfoOrClosedOrConfigFileInfo): infoOrFileNameOrConfig is AncestorConfigFileInfo {
    return !!(infoOrFileNameOrConfig as AncestorConfigFileInfo).configFileInfo;
}

/*@internal*/
/** Kind of operation to perform to get project reference project */
export enum ProjectReferenceProjectLoadKind {
    /** Find existing project for project reference */
    Find,
    /** Find existing project or create one for the project reference */
    FindCreate,
    /** Find existing project or create and load it for the project reference */
    FindCreateLoad
}

/*@internal*/
export function forEachResolvedProjectReferenceProject<T>(
    project: ts.server.ConfiguredProject,
    fileName: string | undefined,
    cb: (child: ts.server.ConfiguredProject) => T | undefined,
    projectReferenceProjectLoadKind: ProjectReferenceProjectLoadKind.Find | ProjectReferenceProjectLoadKind.FindCreate,
): T | undefined;
/*@internal*/
export function forEachResolvedProjectReferenceProject<T>(
    project: ts.server.ConfiguredProject,
    fileName: string | undefined,
    cb: (child: ts.server.ConfiguredProject) => T | undefined,
    projectReferenceProjectLoadKind: ProjectReferenceProjectLoadKind,
    reason: string
): T | undefined;
export function forEachResolvedProjectReferenceProject<T>(
    project: ts.server.ConfiguredProject,
    fileName: string | undefined,
    cb: (child: ts.server.ConfiguredProject) => T | undefined,
    projectReferenceProjectLoadKind: ProjectReferenceProjectLoadKind,
    reason?: string
): T | undefined {
    const resolvedRefs = project.getCurrentProgram()?.getResolvedProjectReferences();
    if (!resolvedRefs) return undefined;
    let seenResolvedRefs: ts.ESMap<string, ProjectReferenceProjectLoadKind> | undefined;
    const possibleDefaultRef = fileName ? project.getResolvedProjectReferenceToRedirect(fileName) : undefined;
    if (possibleDefaultRef) {
        // Try to find the name of the file directly through resolved project references
        const configFileName = ts.server.toNormalizedPath(possibleDefaultRef.sourceFile.fileName);
        const child = project.projectService.findConfiguredProjectByProjectName(configFileName);
        if (child) {
            const result = cb(child);
            if (result) return result;
        }
        else if (projectReferenceProjectLoadKind !== ProjectReferenceProjectLoadKind.Find) {
            seenResolvedRefs = new ts.Map();
            // Try to see if this project can be loaded
            const result = forEachResolvedProjectReferenceProjectWorker(
                resolvedRefs,
                project.getCompilerOptions(),
                (ref, loadKind) => possibleDefaultRef === ref ? callback(ref, loadKind) : undefined,
                projectReferenceProjectLoadKind,
                project.projectService,
                seenResolvedRefs
            );
            if (result) return result;
            // Cleanup seenResolvedRefs
            seenResolvedRefs.clear();
        }
    }

    return forEachResolvedProjectReferenceProjectWorker(
        resolvedRefs,
        project.getCompilerOptions(),
        (ref, loadKind) => possibleDefaultRef !== ref ? callback(ref, loadKind) : undefined,
        projectReferenceProjectLoadKind,
        project.projectService,
        seenResolvedRefs
    );

    function callback(ref: ts.ResolvedProjectReference, loadKind: ProjectReferenceProjectLoadKind) {
        const configFileName = ts.server.toNormalizedPath(ref.sourceFile.fileName);
        const child = project.projectService.findConfiguredProjectByProjectName(configFileName) || (
            loadKind === ProjectReferenceProjectLoadKind.Find ?
                undefined :
                loadKind === ProjectReferenceProjectLoadKind.FindCreate ?
                    project.projectService.createConfiguredProject(configFileName) :
                    loadKind === ProjectReferenceProjectLoadKind.FindCreateLoad ?
                        project.projectService.createAndLoadConfiguredProject(configFileName, reason!) :
                        ts.Debug.assertNever(loadKind)
        );

        return child && cb(child);
    }
}

function forEachResolvedProjectReferenceProjectWorker<T>(
    resolvedProjectReferences: readonly (ts.ResolvedProjectReference | undefined)[],
    parentOptions: ts.CompilerOptions,
    cb: (resolvedRef: ts.ResolvedProjectReference, loadKind: ProjectReferenceProjectLoadKind) => T | undefined,
    projectReferenceProjectLoadKind: ProjectReferenceProjectLoadKind,
    projectService: ProjectService,
    seenResolvedRefs: ts.ESMap<string, ProjectReferenceProjectLoadKind> | undefined,
): T | undefined {
    const loadKind = parentOptions.disableReferencedProjectLoad ? ProjectReferenceProjectLoadKind.Find : projectReferenceProjectLoadKind;
    return ts.forEach(resolvedProjectReferences, ref => {
        if (!ref) return undefined;

        const configFileName = ts.server.toNormalizedPath(ref.sourceFile.fileName);
        const canonicalPath = projectService.toCanonicalFileName(configFileName);
        const seenValue = seenResolvedRefs?.get(canonicalPath);
        if (seenValue !== undefined && seenValue >= loadKind) {
            return undefined;
        }
        const result = cb(ref, loadKind);
        if (result) {
            return result;
        }

        (seenResolvedRefs || (seenResolvedRefs = new ts.Map())).set(canonicalPath, loadKind);
        return ref.references && forEachResolvedProjectReferenceProjectWorker(ref.references, ref.commandLine.options, cb, loadKind, projectService, seenResolvedRefs);
    });
}

function forEachPotentialProjectReference<T>(
    project: ts.server.ConfiguredProject,
    cb: (potentialProjectReference: ts.Path) => T | undefined
): T | undefined {
    return project.potentialProjectReferences &&
        ts.forEachKey(project.potentialProjectReferences, cb);
}

function forEachAnyProjectReferenceKind<T>(
    project: ts.server.ConfiguredProject,
    cb: (resolvedProjectReference: ts.ResolvedProjectReference) => T | undefined,
    cbProjectRef: (projectReference: ts.ProjectReference) => T | undefined,
    cbPotentialProjectRef: (potentialProjectReference: ts.Path) => T | undefined
): T | undefined {
    return project.getCurrentProgram() ?
        project.forEachResolvedProjectReference(cb) :
        project.isInitialLoadPending() ?
            forEachPotentialProjectReference(project, cbPotentialProjectRef) :
            ts.forEach(project.getProjectReferences(), cbProjectRef);
}

function callbackRefProject<T>(
    project: ts.server.ConfiguredProject,
    cb: (refProj: ts.server.ConfiguredProject) => T | undefined,
    refPath: ts.Path | undefined
) {
    const refProject = refPath && project.projectService.configuredProjects.get(refPath);
    return refProject && cb(refProject);
}

function forEachReferencedProject<T>(
    project: ts.server.ConfiguredProject,
    cb: (refProj: ts.server.ConfiguredProject) => T | undefined
): T | undefined {
    return forEachAnyProjectReferenceKind(
        project,
        resolvedRef => callbackRefProject(project, cb, resolvedRef.sourceFile.path),
        projectRef => callbackRefProject(project, cb, project.toPath(ts.resolveProjectReferencePath(projectRef))),
        potentialProjectRef => callbackRefProject(project, cb, potentialProjectRef)
    );
}

interface NodeModulesWatcher extends ts.FileWatcher {
    /** How many watchers of this directory were for closed ScriptInfo */
    refreshScriptInfoRefCount: number;
    /** List of project names whose module specifier cache should be cleared when package.jsons change */
    affectedModuleSpecifierCacheProjects?: ts.Set<string>;
}

function getDetailWatchInfo(watchType: ts.WatchType, project: ts.server.Project | ts.server.NormalizedPath | undefined) {
    return `${ts.isString(project) ? `Config: ${project} ` : project ? `Project: ${project.getProjectName()} ` : ""}WatchType: ${watchType}`;
}

function isScriptInfoWatchedFromNodeModules(info: ts.server.ScriptInfo) {
    return !info.isScriptOpen() && info.mTime !== undefined;
}

/*@internal*/
/** true if script info is part of project and is not in project because it is referenced from project reference source */
export function projectContainsInfoDirectly(project: ts.server.Project, info: ts.server.ScriptInfo) {
    return project.containsScriptInfo(info) &&
        !project.isSourceOfProjectReferenceRedirect(info.path);
}

/*@internal*/
export function updateProjectIfDirty(project: ts.server.Project) {
    project.invalidateResolutionsOfFailedLookupLocations();
    return project.dirty && project.updateGraph();
}

function setProjectOptionsUsed(project: ts.server.ConfiguredProject | ts.server.ExternalProject) {
    if (ts.server.isConfiguredProject(project)) {
        project.projectOptions = true;
    }
}

/*@internal*/
export interface OpenFileArguments {
    fileName: string;
    content?: string;
    scriptKind?: ts.server.protocol.ScriptKindName | ts.ScriptKind;
    hasMixedContent?: boolean;
    projectRootPath?: string;
}

/*@internal*/
export interface ChangeFileArguments {
    fileName: string;
    changes: ts.Iterator<ts.TextChange>;
}

export interface WatchOptionsAndErrors {
    watchOptions: ts.WatchOptions;
    errors: ts.Diagnostic[] | undefined;
}

/*@internal*/
export interface ParsedConfig{
    cachedDirectoryStructureHost: ts.CachedDirectoryStructureHost;
    /**
     * The map contains
     *   - true if project is watching config file as well as wild cards
     *   - false if just config file is watched
     */
    projects: ts.ESMap<ts.server.NormalizedPath, boolean>;
    parsedCommandLine?: ts.ParsedCommandLine;
    watchedDirectories?: ts.Map<ts.WildcardDirectoryWatcher>;
    /**
     * true if watchedDirectories need to be updated as per parsedCommandLine's updated watched directories
     */
    watchedDirectoriesStale?: boolean;
    reloadLevel?: ts.ConfigFileProgramReloadLevel.Partial | ts.ConfigFileProgramReloadLevel.Full;
}

function createProjectNameFactoryWithCounter(nameFactory: (counter: number) => string) {
    let nextId = 1;
    return () => nameFactory(nextId++);
}

export class ProjectService {

    /*@internal*/
    readonly typingsCache: ts.server.TypingsCache;

    /*@internal*/
    readonly documentRegistry: ts.DocumentRegistry;

    /**
     * Container of all known scripts
     */
    /*@internal*/
    readonly filenameToScriptInfo = new ts.Map<string, ts.server.ScriptInfo>();
    private readonly nodeModulesWatchers = new ts.Map<string, NodeModulesWatcher>();
    /**
     * Contains all the deleted script info's version information so that
     * it does not reset when creating script info again
     * (and could have potentially collided with version where contents mismatch)
     */
    private readonly filenameToScriptInfoVersion = new ts.Map<string, ts.server.ScriptInfoVersion>();
    // Set of all '.js' files ever opened.
    private readonly allJsFilesForOpenFileTelemetry = new ts.Map<string, true>();

    /**
     * Map to the real path of the infos
     */
    /* @internal */
    readonly realpathToScriptInfos: ts.MultiMap<ts.Path, ts.server.ScriptInfo> | undefined;
    /**
     * maps external project file name to list of config files that were the part of this project
     */
    private readonly externalProjectToConfiguredProjectMap = new ts.Map<string, ts.server.NormalizedPath[]>();

    /**
     * external projects (configuration and list of root files is not controlled by tsserver)
     */
    readonly externalProjects: ts.server.ExternalProject[] = [];
    /**
     * projects built from openFileRoots
     */
    readonly inferredProjects: ts.server.InferredProject[] = [];
    /**
     * projects specified by a tsconfig.json file
     */
    readonly configuredProjects: ts.Map<ts.server.ConfiguredProject> = new ts.Map<string, ts.server.ConfiguredProject>();
    /*@internal*/
    readonly newInferredProjectName = createProjectNameFactoryWithCounter(ts.server.makeInferredProjectName);
    /*@internal*/
    readonly newAutoImportProviderProjectName = createProjectNameFactoryWithCounter(ts.server.makeAutoImportProviderProjectName);
    /*@internal*/
    readonly newAuxiliaryProjectName = createProjectNameFactoryWithCounter(ts.server.makeAuxiliaryProjectName);
    /**
     * Open files: with value being project root path, and key being Path of the file that is open
     */
    readonly openFiles: ts.Map<ts.server.NormalizedPath | undefined> = new ts.Map<ts.Path, ts.server.NormalizedPath | undefined>();
    /* @internal */
    readonly configFileForOpenFiles: ts.ESMap<ts.Path, ts.server.NormalizedPath | false> = new ts.Map();
    /**
     * Map of open files that are opened without complete path but have projectRoot as current directory
     */
    private readonly openFilesWithNonRootedDiskPath = new ts.Map<string, ts.server.ScriptInfo>();

    private compilerOptionsForInferredProjects: ts.CompilerOptions | undefined;
    private compilerOptionsForInferredProjectsPerProjectRoot = new ts.Map<string, ts.CompilerOptions>();
    private watchOptionsForInferredProjects: WatchOptionsAndErrors | undefined;
    private watchOptionsForInferredProjectsPerProjectRoot = new ts.Map<string, WatchOptionsAndErrors | false>();
    private typeAcquisitionForInferredProjects: ts.TypeAcquisition | undefined;
    private typeAcquisitionForInferredProjectsPerProjectRoot = new ts.Map<string, ts.TypeAcquisition | undefined>();
    /**
     * Project size for configured or external projects
     */
    private readonly projectToSizeMap = new ts.Map<string, number>();
    /**
     * This is a map of config file paths existence that doesnt need query to disk
     * - The entry can be present because there is inferred project that needs to watch addition of config file to directory
     *   In this case the exists could be true/false based on config file is present or not
     * - Or it is present if we have configured project open with config file at that location
     *   In this case the exists property is always true
     */
    /*@internal*/ readonly configFileExistenceInfoCache = new ts.Map<ts.server.NormalizedPath, ConfigFileExistenceInfo>();
    /*@internal*/ readonly throttledOperations: ts.server.ThrottledOperations;

    private readonly hostConfiguration: HostConfiguration;
    private safelist: SafeList = defaultTypeSafeList;
    private readonly legacySafelist = new ts.Map<string, string>();

    private pendingProjectUpdates = new ts.Map<string, ts.server.Project>();
    /* @internal */
    pendingEnsureProjectForOpenFiles = false;

    readonly currentDirectory: ts.server.NormalizedPath;
    readonly toCanonicalFileName: (f: string) => string;

    public readonly host: ts.server.ServerHost;
    public readonly logger: ts.server.Logger;
    public readonly cancellationToken: ts.HostCancellationToken;
    public readonly useSingleInferredProject: boolean;
    public readonly useInferredProjectPerProjectRoot: boolean;
    public readonly typingsInstaller: ts.server.ITypingsInstaller;
    private readonly globalCacheLocationDirectoryPath: ts.Path | undefined;
    public readonly throttleWaitMilliseconds?: number;
    private readonly eventHandler?: ProjectServiceEventHandler;
    private readonly suppressDiagnosticEvents?: boolean;

    public readonly globalPlugins: readonly string[];
    public readonly pluginProbeLocations: readonly string[];
    public readonly allowLocalPluginLoads: boolean;
    private currentPluginConfigOverrides: ts.ESMap<string, any> | undefined;

    public readonly typesMapLocation: string | undefined;

    /** @deprecated use serverMode instead */
    public readonly syntaxOnly: boolean;
    public readonly serverMode: ts.LanguageServiceMode;

    /** Tracks projects that we have already sent telemetry for. */
    private readonly seenProjects = new ts.Map<string, true>();

    /*@internal*/
    readonly watchFactory: ts.WatchFactory<ts.WatchType, ts.server.Project | ts.server.NormalizedPath>;

    /*@internal*/
    private readonly sharedExtendedConfigFileWatchers = new ts.Map<ts.Path, ts.SharedExtendedConfigFileWatcher<ts.server.NormalizedPath>>();
    /*@internal*/
    private readonly extendedConfigCache = new ts.Map<string, ts.ExtendedConfigCacheEntry>();

    /*@internal*/
    readonly packageJsonCache: ts.server.PackageJsonCache;
    /*@internal*/
    private packageJsonFilesMap: ts.ESMap<ts.Path, ts.FileWatcher> | undefined;
    /*@internal*/
    private incompleteCompletionsCache: ts.IncompleteCompletionsCache | undefined;
    /*@internal*/
    readonly session: ts.server.Session<unknown> | undefined;


    private performanceEventHandler?: PerformanceEventHandler;

    private pendingPluginEnablements?: ts.ESMap<ts.server.Project, Promise<ts.server.BeginEnablePluginResult>[]>;
    private currentPluginEnablementPromise?: Promise<void>;

    constructor(opts: ProjectServiceOptions) {
        this.host = opts.host;
        this.logger = opts.logger;
        this.cancellationToken = opts.cancellationToken;
        this.useSingleInferredProject = opts.useSingleInferredProject;
        this.useInferredProjectPerProjectRoot = opts.useInferredProjectPerProjectRoot;
        this.typingsInstaller = opts.typingsInstaller || ts.server.nullTypingsInstaller;
        this.throttleWaitMilliseconds = opts.throttleWaitMilliseconds;
        this.eventHandler = opts.eventHandler;
        this.suppressDiagnosticEvents = opts.suppressDiagnosticEvents;
        this.globalPlugins = opts.globalPlugins || ts.server.emptyArray;
        this.pluginProbeLocations = opts.pluginProbeLocations || ts.server.emptyArray;
        this.allowLocalPluginLoads = !!opts.allowLocalPluginLoads;
        this.typesMapLocation = (opts.typesMapLocation === undefined) ? ts.combinePaths(ts.getDirectoryPath(this.getExecutingFilePath()), "typesMap.json") : opts.typesMapLocation;
        this.session = opts.session;

        if (opts.serverMode !== undefined) {
            this.serverMode = opts.serverMode;
            this.syntaxOnly = this.serverMode === ts.LanguageServiceMode.Syntactic;
        }
        else if (opts.syntaxOnly) {
            this.serverMode = ts.LanguageServiceMode.Syntactic;
            this.syntaxOnly = true;
        }
        else {
            this.serverMode = ts.LanguageServiceMode.Semantic;
            this.syntaxOnly = false;
        }

        if (this.host.realpath) {
            this.realpathToScriptInfos = ts.createMultiMap();
        }
        this.currentDirectory = ts.server.toNormalizedPath(this.host.getCurrentDirectory());
        this.toCanonicalFileName = ts.createGetCanonicalFileName(this.host.useCaseSensitiveFileNames);
        this.globalCacheLocationDirectoryPath = this.typingsInstaller.globalTypingsCacheLocation
            ? ts.ensureTrailingDirectorySeparator(this.toPath(this.typingsInstaller.globalTypingsCacheLocation))
            : undefined;
        this.throttledOperations = new ts.server.ThrottledOperations(this.host, this.logger);

        if (this.typesMapLocation) {
            this.loadTypesMap();
        }
        else {
            this.logger.info("No types map provided; using the default");
        }

        this.typingsInstaller.attach(this);

        this.typingsCache = new ts.server.TypingsCache(this.typingsInstaller);

        this.hostConfiguration = {
            formatCodeOptions: ts.getDefaultFormatCodeSettings(this.host.newLine),
            preferences: ts.emptyOptions,
            hostInfo: "Unknown host",
            extraFileExtensions: [],
        };

        this.documentRegistry = ts.createDocumentRegistryInternal(this.host.useCaseSensitiveFileNames, this.currentDirectory, this);
        const watchLogLevel = this.logger.hasLevel(ts.server.LogLevel.verbose) ? ts.WatchLogLevel.Verbose :
            this.logger.loggingEnabled() ? ts.WatchLogLevel.TriggerOnly : ts.WatchLogLevel.None;
        const log: (s: string) => void = watchLogLevel !== ts.WatchLogLevel.None ? (s => this.logger.info(s)) : ts.noop;
        this.packageJsonCache = ts.server.createPackageJsonCache(this);
        this.watchFactory = this.serverMode !== ts.LanguageServiceMode.Semantic ?
            {
                watchFile: ts.returnNoopFileWatcher,
                watchDirectory: ts.returnNoopFileWatcher,
            } :
            ts.getWatchFactory(this.host, watchLogLevel, log, getDetailWatchInfo);
    }

    toPath(fileName: string) {
        return ts.toPath(fileName, this.currentDirectory, this.toCanonicalFileName);
    }

    /*@internal*/
    getExecutingFilePath() {
        return this.getNormalizedAbsolutePath(this.host.getExecutingFilePath());
    }

    /*@internal*/
    getNormalizedAbsolutePath(fileName: string) {
        return ts.getNormalizedAbsolutePath(fileName, this.host.getCurrentDirectory());
    }

    /*@internal*/
    setDocument(key: ts.DocumentRegistryBucketKeyWithMode, path: ts.Path, sourceFile: ts.SourceFile) {
        const info = ts.Debug.checkDefined(this.getScriptInfoForPath(path));
        info.cacheSourceFile = { key, sourceFile };
    }

    /*@internal*/
    getDocument(key: ts.DocumentRegistryBucketKeyWithMode, path: ts.Path): ts.SourceFile | undefined {
        const info = this.getScriptInfoForPath(path);
        return info && info.cacheSourceFile && info.cacheSourceFile.key === key ? info.cacheSourceFile.sourceFile : undefined;
    }

    /* @internal */
    ensureInferredProjectsUpToDate_TestOnly() {
        this.ensureProjectStructuresUptoDate();
    }

    /* @internal */
    getCompilerOptionsForInferredProjects() {
        return this.compilerOptionsForInferredProjects;
    }

    /* @internal */
    onUpdateLanguageServiceStateForProject(project: ts.server.Project, languageServiceEnabled: boolean) {
        if (!this.eventHandler) {
            return;
        }
        const event: ProjectLanguageServiceStateEvent = {
            eventName: ProjectLanguageServiceStateEvent,
            data: { project, languageServiceEnabled }
        };
        this.eventHandler(event);
    }

    private loadTypesMap() {
        try {
            const fileContent = this.host.readFile(this.typesMapLocation!); // TODO: GH#18217
            if (fileContent === undefined) {
                this.logger.info(`Provided types map file "${this.typesMapLocation}" doesn't exist`);
                return;
            }
            const raw: TypesMapFile = JSON.parse(fileContent);
            // Parse the regexps
            for (const k of Object.keys(raw.typesMap)) {
                raw.typesMap[k].match = new RegExp(raw.typesMap[k].match as {} as string, "i");
            }
            // raw is now fixed and ready
            this.safelist = raw.typesMap;
            for (const key in raw.simpleMap) {
                if (ts.hasProperty(raw.simpleMap, key)) {
                    this.legacySafelist.set(key, raw.simpleMap[key].toLowerCase());
                }
            }
        }
        catch (e) {
            this.logger.info(`Error loading types map: ${e}`);
            this.safelist = defaultTypeSafeList;
            this.legacySafelist.clear();
        }
    }

    updateTypingsForProject(response: ts.server.SetTypings | ts.server.InvalidateCachedTypings | ts.server.PackageInstalledResponse): void;
    /** @internal */
    updateTypingsForProject(response: ts.server.SetTypings | ts.server.InvalidateCachedTypings | ts.server.PackageInstalledResponse | ts.server.BeginInstallTypes | ts.server.EndInstallTypes): void; // eslint-disable-line @typescript-eslint/unified-signatures
    updateTypingsForProject(response: ts.server.SetTypings | ts.server.InvalidateCachedTypings | ts.server.PackageInstalledResponse | ts.server.BeginInstallTypes | ts.server.EndInstallTypes): void {
        const project = this.findProject(response.projectName);
        if (!project) {
            return;
        }
        switch (response.kind) {
            case ts.server.ActionSet:
                // Update the typing files and update the project
                project.updateTypingFiles(this.typingsCache.updateTypingsForProject(response.projectName, response.compilerOptions, response.typeAcquisition, response.unresolvedImports, response.typings));
                return;
            case ts.server.ActionInvalidate:
                // Do not clear resolution cache, there was changes detected in typings, so enque typing request and let it get us correct results
                this.typingsCache.enqueueInstallTypingsForProject(project, project.lastCachedUnresolvedImportsList, /*forceRefresh*/ true);
                return;
        }
    }

    /*@internal*/
    delayEnsureProjectForOpenFiles() {
        if (!this.openFiles.size) return;
        this.pendingEnsureProjectForOpenFiles = true;
        this.throttledOperations.schedule(ensureProjectForOpenFileSchedule, /*delay*/ 2500, () => {
            if (this.pendingProjectUpdates.size !== 0) {
                this.delayEnsureProjectForOpenFiles();
            }
            else {
                if (this.pendingEnsureProjectForOpenFiles) {
                    this.ensureProjectForOpenFiles();

                    // Send the event to notify that there were background project updates
                    // send current list of open files
                    this.sendProjectsUpdatedInBackgroundEvent();
                }
            }
        });
    }

    private delayUpdateProjectGraph(project: ts.server.Project) {
        project.markAsDirty();
        if (project.projectKind !== ts.server.ProjectKind.AutoImportProvider && project.projectKind !== ts.server.ProjectKind.Auxiliary) {
            const projectName = project.getProjectName();
            this.pendingProjectUpdates.set(projectName, project);
            this.throttledOperations.schedule(projectName, /*delay*/ 250, () => {
                if (this.pendingProjectUpdates.delete(projectName)) {
                    updateProjectIfDirty(project);
                }
            });
        }
    }

    /*@internal*/
    hasPendingProjectUpdate(project: ts.server.Project) {
        return this.pendingProjectUpdates.has(project.getProjectName());
    }

    /* @internal */
    sendProjectsUpdatedInBackgroundEvent() {
        if (!this.eventHandler) {
            return;
        }

        const event: ProjectsUpdatedInBackgroundEvent = {
            eventName: ProjectsUpdatedInBackgroundEvent,
            data: {
                openFiles: ts.arrayFrom(this.openFiles.keys(), path => this.getScriptInfoForPath(path as ts.Path)!.fileName)
            }
        };
        this.eventHandler(event);
    }

    /* @internal */
    sendLargeFileReferencedEvent(file: string, fileSize: number) {
        if (!this.eventHandler) {
            return;
        }

        const event: LargeFileReferencedEvent = {
            eventName: LargeFileReferencedEvent,
            data: { file, fileSize, maxFileSize }
        };
        this.eventHandler(event);
    }

    /* @internal */
    sendProjectLoadingStartEvent(project: ts.server.ConfiguredProject, reason: string) {
        if (!this.eventHandler) {
            return;
        }
        project.sendLoadingProjectFinish = true;
        const event: ProjectLoadingStartEvent = {
            eventName: ProjectLoadingStartEvent,
            data: { project, reason }
        };
        this.eventHandler(event);
    }

    /* @internal */
    sendProjectLoadingFinishEvent(project: ts.server.ConfiguredProject) {
        if (!this.eventHandler || !project.sendLoadingProjectFinish) {
            return;
        }

        project.sendLoadingProjectFinish = false;
        const event: ProjectLoadingFinishEvent = {
            eventName: ProjectLoadingFinishEvent,
            data: { project }
        };
        this.eventHandler(event);
    }

    /* @internal */
    sendPerformanceEvent(kind: ts.PerformanceEvent["kind"], durationMs: number) {
        if (this.performanceEventHandler) {
            this.performanceEventHandler({ kind, durationMs });
        }
    }

    /* @internal */
    delayUpdateProjectGraphAndEnsureProjectStructureForOpenFiles(project: ts.server.Project) {
        this.delayUpdateProjectGraph(project);
        this.delayEnsureProjectForOpenFiles();
    }

    private delayUpdateProjectGraphs(projects: readonly ts.server.Project[], clearSourceMapperCache: boolean) {
        if (projects.length) {
            for (const project of projects) {
                // Even if program doesnt change, clear the source mapper cache
                if (clearSourceMapperCache) project.clearSourceMapperCache();
                this.delayUpdateProjectGraph(project);
            }
            this.delayEnsureProjectForOpenFiles();
        }
    }

    setCompilerOptionsForInferredProjects(projectCompilerOptions: ts.server.protocol.InferredProjectCompilerOptions, projectRootPath?: string): void {
        ts.Debug.assert(projectRootPath === undefined || this.useInferredProjectPerProjectRoot, "Setting compiler options per project root path is only supported when useInferredProjectPerProjectRoot is enabled");

        const compilerOptions = convertCompilerOptions(projectCompilerOptions);
        const watchOptions = convertWatchOptions(projectCompilerOptions, projectRootPath);
        const typeAcquisition = convertTypeAcquisition(projectCompilerOptions);

        // always set 'allowNonTsExtensions' for inferred projects since user cannot configure it from the outside
        // previously we did not expose a way for user to change these settings and this option was enabled by default
        compilerOptions.allowNonTsExtensions = true;
        const canonicalProjectRootPath = projectRootPath && this.toCanonicalFileName(projectRootPath);
        if (canonicalProjectRootPath) {
            this.compilerOptionsForInferredProjectsPerProjectRoot.set(canonicalProjectRootPath, compilerOptions);
            this.watchOptionsForInferredProjectsPerProjectRoot.set(canonicalProjectRootPath, watchOptions || false);
            this.typeAcquisitionForInferredProjectsPerProjectRoot.set(canonicalProjectRootPath, typeAcquisition);
        }
        else {
            this.compilerOptionsForInferredProjects = compilerOptions;
            this.watchOptionsForInferredProjects = watchOptions;
            this.typeAcquisitionForInferredProjects = typeAcquisition;
        }

        for (const project of this.inferredProjects) {
            // Only update compiler options in the following cases:
            // - Inferred projects without a projectRootPath, if the new options do not apply to
            //   a workspace root
            // - Inferred projects with a projectRootPath, if the new options do not apply to a
            //   workspace root and there is no more specific set of options for that project's
            //   root path
            // - Inferred projects with a projectRootPath, if the new options apply to that
            //   project root path.
            if (canonicalProjectRootPath ?
                project.projectRootPath === canonicalProjectRootPath :
                !project.projectRootPath || !this.compilerOptionsForInferredProjectsPerProjectRoot.has(project.projectRootPath)) {
                project.setCompilerOptions(compilerOptions);
                project.setTypeAcquisition(typeAcquisition);
                project.setWatchOptions(watchOptions?.watchOptions);
                project.setProjectErrors(watchOptions?.errors);
                project.compileOnSaveEnabled = compilerOptions.compileOnSave!;
                project.markAsDirty();
                this.delayUpdateProjectGraph(project);
            }
        }

        this.delayEnsureProjectForOpenFiles();
    }

    findProject(projectName: string): ts.server.Project | undefined {
        if (projectName === undefined) {
            return undefined;
        }
        if (ts.server.isInferredProjectName(projectName)) {
            return findProjectByName(projectName, this.inferredProjects);
        }
        return this.findExternalProjectByProjectName(projectName) || this.findConfiguredProjectByProjectName(ts.server.toNormalizedPath(projectName));
    }

    /* @internal */
    private forEachProject(cb: (project: ts.server.Project) => void) {
        this.externalProjects.forEach(cb);
        this.configuredProjects.forEach(cb);
        this.inferredProjects.forEach(cb);
    }

    /* @internal */
    forEachEnabledProject(cb: (project: ts.server.Project) => void) {
        this.forEachProject(project => {
            if (!project.isOrphan() && project.languageServiceEnabled) {
                cb(project);
            }
        });
    }

    getDefaultProjectForFile(fileName: ts.server.NormalizedPath, ensureProject: boolean): ts.server.Project | undefined {
        return ensureProject ? this.ensureDefaultProjectForFile(fileName) : this.tryGetDefaultProjectForFile(fileName);
    }

    /* @internal */
    tryGetDefaultProjectForFile(fileNameOrScriptInfo: ts.server.NormalizedPath | ts.server.ScriptInfo): ts.server.Project | undefined {
        const scriptInfo = ts.isString(fileNameOrScriptInfo) ? this.getScriptInfoForNormalizedPath(fileNameOrScriptInfo) : fileNameOrScriptInfo;
        return scriptInfo && !scriptInfo.isOrphan() ? scriptInfo.getDefaultProject() : undefined;
    }

    /* @internal */
    ensureDefaultProjectForFile(fileNameOrScriptInfo: ts.server.NormalizedPath | ts.server.ScriptInfo): ts.server.Project {
        return this.tryGetDefaultProjectForFile(fileNameOrScriptInfo) || this.doEnsureDefaultProjectForFile(fileNameOrScriptInfo);
    }

    private doEnsureDefaultProjectForFile(fileNameOrScriptInfo: ts.server.NormalizedPath | ts.server.ScriptInfo): ts.server.Project {
        this.ensureProjectStructuresUptoDate();
        const scriptInfo = ts.isString(fileNameOrScriptInfo) ? this.getScriptInfoForNormalizedPath(fileNameOrScriptInfo) : fileNameOrScriptInfo;
        return scriptInfo ?
            scriptInfo.getDefaultProject() :
            (this.logErrorForScriptInfoNotFound(ts.isString(fileNameOrScriptInfo) ? fileNameOrScriptInfo : fileNameOrScriptInfo.fileName), ts.server.Errors.ThrowNoProject());
    }

    getScriptInfoEnsuringProjectsUptoDate(uncheckedFileName: string) {
        this.ensureProjectStructuresUptoDate();
        return this.getScriptInfo(uncheckedFileName);
    }

    /**
     * Ensures the project structures are upto date
     * This means,
     * - we go through all the projects and update them if they are dirty
     * - if updates reflect some change in structure or there was pending request to ensure projects for open files
     *   ensure that each open script info has project
     */
    private ensureProjectStructuresUptoDate() {
        let hasChanges = this.pendingEnsureProjectForOpenFiles;
        this.pendingProjectUpdates.clear();
        const updateGraph = (project: ts.server.Project) => {
            hasChanges = updateProjectIfDirty(project) || hasChanges;
        };

        this.externalProjects.forEach(updateGraph);
        this.configuredProjects.forEach(updateGraph);
        this.inferredProjects.forEach(updateGraph);
        if (hasChanges) {
            this.ensureProjectForOpenFiles();
        }
    }

    getFormatCodeOptions(file: ts.server.NormalizedPath) {
        const info = this.getScriptInfoForNormalizedPath(file);
        return info && info.getFormatCodeSettings() || this.hostConfiguration.formatCodeOptions;
    }

    getPreferences(file: ts.server.NormalizedPath): ts.server.protocol.UserPreferences {
        const info = this.getScriptInfoForNormalizedPath(file);
        return { ...this.hostConfiguration.preferences, ...info && info.getPreferences() };
    }

    getHostFormatCodeOptions(): ts.FormatCodeSettings {
        return this.hostConfiguration.formatCodeOptions;
    }

    getHostPreferences(): ts.server.protocol.UserPreferences {
        return this.hostConfiguration.preferences;
    }

    private onSourceFileChanged(info: ts.server.ScriptInfo, eventKind: ts.FileWatcherEventKind) {
        if (eventKind === ts.FileWatcherEventKind.Deleted) {
            // File was deleted
            this.handleDeletedFile(info);
        }
        else if (!info.isScriptOpen()) {
            // file has been changed which might affect the set of referenced files in projects that include
            // this file and set of inferred projects
            info.delayReloadNonMixedContentFile();
            this.delayUpdateProjectGraphs(info.containingProjects, /*clearSourceMapperCache*/ false);
            this.handleSourceMapProjects(info);
        }
    }

    private handleSourceMapProjects(info: ts.server.ScriptInfo) {
        // Change in d.ts, update source projects as well
        if (info.sourceMapFilePath) {
            if (ts.isString(info.sourceMapFilePath)) {
                const sourceMapFileInfo = this.getScriptInfoForPath(info.sourceMapFilePath);
                this.delayUpdateSourceInfoProjects(sourceMapFileInfo && sourceMapFileInfo.sourceInfos);
            }
            else {
                this.delayUpdateSourceInfoProjects(info.sourceMapFilePath.sourceInfos);
            }
        }
        // Change in mapInfo, update declarationProjects and source projects
        this.delayUpdateSourceInfoProjects(info.sourceInfos);
        if (info.declarationInfoPath) {
            this.delayUpdateProjectsOfScriptInfoPath(info.declarationInfoPath);
        }
    }

    private delayUpdateSourceInfoProjects(sourceInfos: ts.Set<ts.Path> | undefined) {
        if (sourceInfos) {
            sourceInfos.forEach((_value, path) => this.delayUpdateProjectsOfScriptInfoPath(path));
        }
    }

    private delayUpdateProjectsOfScriptInfoPath(path: ts.Path) {
        const info = this.getScriptInfoForPath(path);
        if (info) {
            this.delayUpdateProjectGraphs(info.containingProjects, /*clearSourceMapperCache*/ true);
        }
    }

    private handleDeletedFile(info: ts.server.ScriptInfo) {
        this.stopWatchingScriptInfo(info);

        if (!info.isScriptOpen()) {
            this.deleteScriptInfo(info);

            // capture list of projects since detachAllProjects will wipe out original list
            const containingProjects = info.containingProjects.slice();

            info.detachAllProjects();

            // update projects to make sure that set of referenced files is correct
            this.delayUpdateProjectGraphs(containingProjects, /*clearSourceMapperCache*/ false);
            this.handleSourceMapProjects(info);
            info.closeSourceMapFileWatcher();
            // need to recalculate source map from declaration file
            if (info.declarationInfoPath) {
                const declarationInfo = this.getScriptInfoForPath(info.declarationInfoPath);
                if (declarationInfo) {
                    declarationInfo.sourceMapFilePath = undefined;
                }
            }
        }
    }

    /**
     * This is to watch whenever files are added or removed to the wildcard directories
     */
    /*@internal*/
    private watchWildcardDirectory(directory: ts.Path, flags: ts.WatchDirectoryFlags, configFileName: ts.server.NormalizedPath, config: ParsedConfig) {
        return this.watchFactory.watchDirectory(
            directory,
            fileOrDirectory => {
                const fileOrDirectoryPath = this.toPath(fileOrDirectory);
                const fsResult = config.cachedDirectoryStructureHost.addOrDeleteFileOrDirectory(fileOrDirectory, fileOrDirectoryPath);
                if (ts.getBaseFileName(fileOrDirectoryPath) === "package.json" && !ts.isInsideNodeModules(fileOrDirectoryPath) &&
                    (fsResult && fsResult.fileExists || !fsResult && this.host.fileExists(fileOrDirectoryPath))
                ) {
                    this.logger.info(`Config: ${configFileName} Detected new package.json: ${fileOrDirectory}`);
                    this.onAddPackageJson(fileOrDirectoryPath);
                }

                const configuredProjectForConfig = this.findConfiguredProjectByProjectName(configFileName);
                if (ts.isIgnoredFileFromWildCardWatching({
                    watchedDirPath: directory,
                    fileOrDirectory,
                    fileOrDirectoryPath,
                    configFileName,
                    extraFileExtensions: this.hostConfiguration.extraFileExtensions,
                    currentDirectory: this.currentDirectory,
                    options: config.parsedCommandLine!.options,
                    program: configuredProjectForConfig?.getCurrentProgram() || config.parsedCommandLine!.fileNames,
                    useCaseSensitiveFileNames: this.host.useCaseSensitiveFileNames,
                    writeLog: s => this.logger.info(s),
                    toPath: s => this.toPath(s)
                })) return;

                // Reload is pending, do the reload
                if (config.reloadLevel !== ts.ConfigFileProgramReloadLevel.Full) config.reloadLevel = ts.ConfigFileProgramReloadLevel.Partial;
                config.projects.forEach((watchWildcardDirectories, projectCanonicalPath) => {
                    if (!watchWildcardDirectories) return;
                    const project = this.getConfiguredProjectByCanonicalConfigFilePath(projectCanonicalPath);
                    if (!project) return;

                    // Load root file names for configured project with the config file name
                    // But only schedule update if project references this config file
                    const reloadLevel = configuredProjectForConfig === project ? ts.ConfigFileProgramReloadLevel.Partial : ts.ConfigFileProgramReloadLevel.None;
                    if (project.pendingReload !== undefined && project.pendingReload > reloadLevel) return;

                    // don't trigger callback on open, existing files
                    if (this.openFiles.has(fileOrDirectoryPath)) {
                        const info = ts.Debug.checkDefined(this.getScriptInfoForPath(fileOrDirectoryPath));
                        if (info.isAttached(project)) {
                            const loadLevelToSet = Math.max(reloadLevel, project.openFileWatchTriggered.get(fileOrDirectoryPath) || ts.ConfigFileProgramReloadLevel.None) as ts.ConfigFileProgramReloadLevel;
                            project.openFileWatchTriggered.set(fileOrDirectoryPath, loadLevelToSet);
                        }
                        else {
                            project.pendingReload = reloadLevel;
                            this.delayUpdateProjectGraphAndEnsureProjectStructureForOpenFiles(project);
                        }
                    }
                    else {
                        project.pendingReload = reloadLevel;
                        this.delayUpdateProjectGraphAndEnsureProjectStructureForOpenFiles(project);
                    }
                });
            },
            flags,
            this.getWatchOptionsFromProjectWatchOptions(config.parsedCommandLine!.watchOptions),
            ts.WatchType.WildcardDirectory,
            configFileName
        );
    }

    /*@internal*/
    private delayUpdateProjectsFromParsedConfigOnConfigFileChange(canonicalConfigFilePath: ts.server.NormalizedPath, reloadReason: string) {
        const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
        if (!configFileExistenceInfo?.config) return false;
        let scheduledAnyProjectUpdate = false;
        // Update projects watching cached config
        configFileExistenceInfo.config.reloadLevel = ts.ConfigFileProgramReloadLevel.Full;

        configFileExistenceInfo.config.projects.forEach((_watchWildcardDirectories, projectCanonicalPath) => {
            const project = this.getConfiguredProjectByCanonicalConfigFilePath(projectCanonicalPath);
            if (!project) return;

            scheduledAnyProjectUpdate = true;
            if (projectCanonicalPath === canonicalConfigFilePath) {
                // Skip refresh if project is not yet loaded
                if (project.isInitialLoadPending()) return;
                project.pendingReload = ts.ConfigFileProgramReloadLevel.Full;
                project.pendingReloadReason = reloadReason;
                this.delayUpdateProjectGraph(project);
            }
            else {
                // Change in referenced project config file
                project.resolutionCache.removeResolutionsFromProjectReferenceRedirects(this.toPath(canonicalConfigFilePath));
                this.delayUpdateProjectGraph(project);
            }
        });
        return scheduledAnyProjectUpdate;
    }

    /*@internal*/
    private onConfigFileChanged(canonicalConfigFilePath: ts.server.NormalizedPath, eventKind: ts.FileWatcherEventKind) {
        const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath)!;
        if (eventKind === ts.FileWatcherEventKind.Deleted) {
            // Update the cached status
            // We arent updating or removing the cached config file presence info as that will be taken care of by
            // releaseParsedConfig when the project is closed or doesnt need this config any more (depending on tracking open files)
            configFileExistenceInfo.exists = false;

            // Remove the configured project for this config file
            const project = configFileExistenceInfo.config?.projects.has(canonicalConfigFilePath) ?
                this.getConfiguredProjectByCanonicalConfigFilePath(canonicalConfigFilePath) :
                undefined;
            if (project) this.removeProject(project);
        }
        else {
            // Update the cached status
            configFileExistenceInfo.exists = true;
        }

        // Update projects watching config
        this.delayUpdateProjectsFromParsedConfigOnConfigFileChange(canonicalConfigFilePath, "Change in config file detected");

        // Reload the configured projects for the open files in the map as they are affected by this config file
        // If the configured project was deleted, we want to reload projects for all the open files including files
        // that are not root of the inferred project
        // Otherwise, we scheduled the update on configured project graph,
        // we would need to schedule the project reload for only the root of inferred projects
        // Get open files to reload projects for
        this.reloadConfiguredProjectForFiles(
            configFileExistenceInfo.openFilesImpactedByConfigFile,
            /*clearSemanticCache*/ false,
            /*delayReload*/ true,
            eventKind !== ts.FileWatcherEventKind.Deleted ?
                ts.identity : // Reload open files if they are root of inferred project
                ts.returnTrue, // Reload all the open files impacted by config file
            "Change in config file detected"
        );
        this.delayEnsureProjectForOpenFiles();
    }

    private removeProject(project: ts.server.Project) {
        this.logger.info("`remove Project::");
        project.print(/*writeProjectFileNames*/ true);

        project.close();
        if (ts.Debug.shouldAssert(ts.AssertionLevel.Normal)) {
            this.filenameToScriptInfo.forEach(info => ts.Debug.assert(
                !info.isAttached(project),
                "Found script Info still attached to project",
                () => `${project.projectName}: ScriptInfos still attached: ${JSON.stringify(
                    ts.arrayFrom(
                        ts.mapDefinedIterator(
                            this.filenameToScriptInfo.values(),
                            info => info.isAttached(project) ?
                                {
                                    fileName: info.fileName,
                                    projects: info.containingProjects.map(p => p.projectName),
                                    hasMixedContent: info.hasMixedContent
                                } : undefined
                        )
                    ),
                    /*replacer*/ undefined,
                    " "
                )}`));
        }
        // Remove the project from pending project updates
        this.pendingProjectUpdates.delete(project.getProjectName());

        switch (project.projectKind) {
            case ts.server.ProjectKind.External:
                ts.unorderedRemoveItem(this.externalProjects, project as ts.server.ExternalProject);
                this.projectToSizeMap.delete(project.getProjectName());
                break;
            case ts.server.ProjectKind.Configured:
                this.configuredProjects.delete((project as ts.server.ConfiguredProject).canonicalConfigFilePath);
                this.projectToSizeMap.delete((project as ts.server.ConfiguredProject).canonicalConfigFilePath);
                break;
            case ts.server.ProjectKind.Inferred:
                ts.unorderedRemoveItem(this.inferredProjects, project as ts.server.InferredProject);
                break;
        }
    }

    /*@internal*/
    assignOrphanScriptInfoToInferredProject(info: ts.server.ScriptInfo, projectRootPath: ts.server.NormalizedPath | undefined) {
        ts.Debug.assert(info.isOrphan());

        const project = this.getOrCreateInferredProjectForProjectRootPathIfEnabled(info, projectRootPath) ||
            this.getOrCreateSingleInferredProjectIfEnabled() ||
            this.getOrCreateSingleInferredWithoutProjectRoot(
                info.isDynamic ?
                    projectRootPath || this.currentDirectory :
                    ts.getDirectoryPath(
                        ts.isRootedDiskPath(info.fileName) ?
                            info.fileName :
                            ts.getNormalizedAbsolutePath(
                                info.fileName,
                                projectRootPath ?
                                    this.getNormalizedAbsolutePath(projectRootPath) :
                                    this.currentDirectory
                            )
                    )
            );

        project.addRoot(info);
        if (info.containingProjects[0] !== project) {
            // Ensure this is first project, we could be in this scenario because info could be part of orphan project
            info.detachFromProject(project);
            info.containingProjects.unshift(project);
        }
        project.updateGraph();

        if (!this.useSingleInferredProject && !project.projectRootPath) {
            // Note that we need to create a copy of the array since the list of project can change
            for (const inferredProject of this.inferredProjects) {
                if (inferredProject === project || inferredProject.isOrphan()) {
                    continue;
                }

                // Remove the inferred project if the root of it is now part of newly created inferred project
                // e.g through references
                // Which means if any root of inferred project is part of more than 1 project can be removed
                // This logic is same as iterating over all open files and calling
                // this.removeRootOfInferredProjectIfNowPartOfOtherProject(f);
                // Since this is also called from refreshInferredProject and closeOpen file
                // to update inferred projects of the open file, this iteration might be faster
                // instead of scanning all open files
                const roots = inferredProject.getRootScriptInfos();
                ts.Debug.assert(roots.length === 1 || !!inferredProject.projectRootPath);
                if (roots.length === 1 && ts.forEach(roots[0].containingProjects, p => p !== roots[0].containingProjects[0] && !p.isOrphan())) {
                    inferredProject.removeFile(roots[0], /*fileExists*/ true, /*detachFromProject*/ true);
                }
            }
        }

        return project;
    }

    private assignOrphanScriptInfosToInferredProject() {
        // collect orphaned files and assign them to inferred project just like we treat open of a file
        this.openFiles.forEach((projectRootPath, path) => {
            const info = this.getScriptInfoForPath(path as ts.Path)!;
            // collect all orphaned script infos from open files
            if (info.isOrphan()) {
                this.assignOrphanScriptInfoToInferredProject(info, projectRootPath);
            }
        });
    }

    /**
     * Remove this file from the set of open, non-configured files.
     * @param info The file that has been closed or newly configured
     */
    private closeOpenFile(info: ts.server.ScriptInfo, skipAssignOrphanScriptInfosToInferredProject?: true) {
        // Closing file should trigger re-reading the file content from disk. This is
        // because the user may chose to discard the buffer content before saving
        // to the disk, and the server's version of the file can be out of sync.
        const fileExists = info.isDynamic ? false : this.host.fileExists(info.fileName);
        info.close(fileExists);
        this.stopWatchingConfigFilesForClosedScriptInfo(info);

        const canonicalFileName = this.toCanonicalFileName(info.fileName);
        if (this.openFilesWithNonRootedDiskPath.get(canonicalFileName) === info) {
            this.openFilesWithNonRootedDiskPath.delete(canonicalFileName);
        }

        // collect all projects that should be removed
        let ensureProjectsForOpenFiles = false;
        for (const p of info.containingProjects) {
            if (ts.server.isConfiguredProject(p)) {
                if (info.hasMixedContent) {
                    info.registerFileUpdate();
                }
                // Do not remove the project so that we can reuse this project
                // if it would need to be re-created with next file open

                // If project had open file affecting
                // Reload the root Files from config if its not already scheduled
                const reloadLevel = p.openFileWatchTriggered.get(info.path);
                if (reloadLevel !== undefined) {
                    p.openFileWatchTriggered.delete(info.path);
                    if (p.pendingReload !== undefined && p.pendingReload < reloadLevel) {
                        p.pendingReload = reloadLevel;
                        p.markFileAsDirty(info.path);
                    }
                }
            }
            else if (ts.server.isInferredProject(p) && p.isRoot(info)) {
                // If this was the last open root file of inferred project
                if (p.isProjectWithSingleRoot()) {
                    ensureProjectsForOpenFiles = true;
                }

                p.removeFile(info, fileExists, /*detachFromProject*/ true);
                // Do not remove the project even if this was last root of the inferred project
                // so that we can reuse this project, if it would need to be re-created with next file open
            }

            if (!p.languageServiceEnabled) {
                // if project language service is disabled then we create a program only for open files.
                // this means that project should be marked as dirty to force rebuilding of the program
                // on the next request
                p.markAsDirty();
            }
        }

        this.openFiles.delete(info.path);
        this.configFileForOpenFiles.delete(info.path);

        if (!skipAssignOrphanScriptInfosToInferredProject && ensureProjectsForOpenFiles) {
            this.assignOrphanScriptInfosToInferredProject();
        }

        // Cleanup script infos that arent part of any project (eg. those could be closed script infos not referenced by any project)
        // is postponed to next file open so that if file from same project is opened,
        // we wont end up creating same script infos

        // If the current info is being just closed - add the watcher file to track changes
        // But if file was deleted, handle that part
        if (fileExists) {
            this.watchClosedScriptInfo(info);
        }
        else {
            this.handleDeletedFile(info);
        }

        return ensureProjectsForOpenFiles;
    }

    private deleteScriptInfo(info: ts.server.ScriptInfo) {
        this.filenameToScriptInfo.delete(info.path);
        this.filenameToScriptInfoVersion.set(info.path, info.getVersion());
        const realpath = info.getRealpathIfDifferent();
        if (realpath) {
            this.realpathToScriptInfos!.remove(realpath, info); // TODO: GH#18217
        }
    }

    private configFileExists(configFileName: ts.server.NormalizedPath, canonicalConfigFilePath: ts.server.NormalizedPath, info: OpenScriptInfoOrClosedOrConfigFileInfo) {
        let configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
        if (configFileExistenceInfo) {
            // By default the info would get impacted by presence of config file since its in the detection path
            // Only adding the info as a root to inferred project will need the existence to be watched by file watcher
            if (isOpenScriptInfo(info) && !configFileExistenceInfo.openFilesImpactedByConfigFile?.has(info.path)) {
                (configFileExistenceInfo.openFilesImpactedByConfigFile ||= new ts.Map()).set(info.path, false);
            }
            return configFileExistenceInfo.exists;
        }

        // Theoretically we should be adding watch for the directory here itself.
        // In practice there will be very few scenarios where the config file gets added
        // somewhere inside the another config file directory.
        // And technically we could handle that case in configFile's directory watcher in some cases
        // But given that its a rare scenario it seems like too much overhead. (we werent watching those directories earlier either)

        // So what we are now watching is: configFile if the configured project corresponding to it is open
        // Or the whole chain of config files for the roots of the inferred projects

        // Cache the host value of file exists and add the info to map of open files impacted by this config file
        const exists = this.host.fileExists(configFileName);
        let openFilesImpactedByConfigFile: ts.ESMap<ts.Path, boolean> | undefined;
        if (isOpenScriptInfo(info)) {
            (openFilesImpactedByConfigFile ||= new ts.Map()).set(info.path, false);
        }
        configFileExistenceInfo = { exists, openFilesImpactedByConfigFile };
        this.configFileExistenceInfoCache.set(canonicalConfigFilePath, configFileExistenceInfo);
        return exists;
    }

    /*@internal*/
    private createConfigFileWatcherForParsedConfig(configFileName: ts.server.NormalizedPath, canonicalConfigFilePath: ts.server.NormalizedPath, forProject: ts.server.ConfiguredProject) {
        const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath)!;
        // When watching config file for parsed config, remove the noopFileWatcher that can be created for open files impacted by config file and watch for real
        if (!configFileExistenceInfo.watcher || configFileExistenceInfo.watcher === noopConfigFileWatcher) {
            configFileExistenceInfo.watcher = this.watchFactory.watchFile(
                configFileName,
                (_fileName, eventKind) => this.onConfigFileChanged(canonicalConfigFilePath, eventKind),
                ts.PollingInterval.High,
                this.getWatchOptionsFromProjectWatchOptions(configFileExistenceInfo?.config?.parsedCommandLine?.watchOptions),
                ts.WatchType.ConfigFile,
                forProject
            );
        }
        // Watching config file for project, update the map
        const projects = configFileExistenceInfo.config!.projects;
        projects.set(forProject.canonicalConfigFilePath, projects.get(forProject.canonicalConfigFilePath) || false);
    }

    /**
     * Returns true if the configFileExistenceInfo is needed/impacted by open files that are root of inferred project
     */
    private configFileExistenceImpactsRootOfInferredProject(configFileExistenceInfo: ConfigFileExistenceInfo) {
        return configFileExistenceInfo.openFilesImpactedByConfigFile &&
            ts.forEachEntry(configFileExistenceInfo.openFilesImpactedByConfigFile, ts.identity);
    }

    /* @internal */
    releaseParsedConfig(canonicalConfigFilePath: ts.server.NormalizedPath, forProject: ts.server.ConfiguredProject) {
        const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath)!;
        if (!configFileExistenceInfo.config?.projects.delete(forProject.canonicalConfigFilePath)) return;
        // If there are still projects watching this config file existence and config, there is nothing to do
        if (configFileExistenceInfo.config?.projects.size) return;

        configFileExistenceInfo.config = undefined;
        ts.clearSharedExtendedConfigFileWatcher(canonicalConfigFilePath, this.sharedExtendedConfigFileWatchers);
        ts.Debug.checkDefined(configFileExistenceInfo.watcher);
        if (configFileExistenceInfo.openFilesImpactedByConfigFile?.size) {
            // If there are open files that are impacted by this config file existence
            // but none of them are root of inferred project, the config file watcher will be
            // created when any of the script infos are added as root of inferred project
            if (this.configFileExistenceImpactsRootOfInferredProject(configFileExistenceInfo)) {
                // If we cannot watch config file existence without configured project, close the configured file watcher
                if (!ts.canWatchDirectoryOrFile(ts.getDirectoryPath(canonicalConfigFilePath) as ts.Path)) {
                    configFileExistenceInfo.watcher!.close();
                    configFileExistenceInfo.watcher = noopConfigFileWatcher;
                }
            }
            else {
                // Close existing watcher
                configFileExistenceInfo.watcher!.close();
                configFileExistenceInfo.watcher = undefined;
            }
        }
        else {
            // There is not a single file open thats tracking the status of this config file. Remove from cache
            configFileExistenceInfo.watcher!.close();
            this.configFileExistenceInfoCache.delete(canonicalConfigFilePath);
        }
    }

    /**
     * Close the config file watcher in the cached ConfigFileExistenceInfo
     *   if there arent any open files that are root of inferred project and there is no parsed config held by any project
     */
    /*@internal*/
    private closeConfigFileWatcherOnReleaseOfOpenFile(configFileExistenceInfo: ConfigFileExistenceInfo) {
        // Close the config file watcher if there are no more open files that are root of inferred project
        // or if there are no projects that need to watch this config file existence info
        if (configFileExistenceInfo.watcher &&
            !configFileExistenceInfo.config &&
            !this.configFileExistenceImpactsRootOfInferredProject(configFileExistenceInfo)) {
            configFileExistenceInfo.watcher.close();
            configFileExistenceInfo.watcher = undefined;
        }
    }

    /**
     * This is called on file close, so that we stop watching the config file for this script info
     */
    private stopWatchingConfigFilesForClosedScriptInfo(info: ts.server.ScriptInfo) {
        ts.Debug.assert(!info.isScriptOpen());
        this.forEachConfigFileLocation(info, canonicalConfigFilePath => {
            const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
            if (configFileExistenceInfo) {
                const infoIsRootOfInferredProject = configFileExistenceInfo.openFilesImpactedByConfigFile?.get(info.path);

                // Delete the info from map, since this file is no more open
                configFileExistenceInfo.openFilesImpactedByConfigFile?.delete(info.path);

                // If the script info was not root of inferred project,
                // there wont be config file watch open because of this script info
                if (infoIsRootOfInferredProject) {
                    // But if it is a root, it could be the last script info that is root of inferred project
                    // and hence we would need to close the config file watcher
                    this.closeConfigFileWatcherOnReleaseOfOpenFile(configFileExistenceInfo);
                }

                // If there are no open files that are impacted by configFileExistenceInfo after closing this script info
                // and there is are no projects that need the config file existence or parsed config,
                // remove the cached existence info
                if (!configFileExistenceInfo.openFilesImpactedByConfigFile?.size &&
                    !configFileExistenceInfo.config) {
                    ts.Debug.assert(!configFileExistenceInfo.watcher);
                    this.configFileExistenceInfoCache.delete(canonicalConfigFilePath);
                }
            }
        });
    }

    /**
     * This is called by inferred project whenever script info is added as a root
     */
    /* @internal */
    startWatchingConfigFilesForInferredProjectRoot(info: ts.server.ScriptInfo) {
        ts.Debug.assert(info.isScriptOpen());
        this.forEachConfigFileLocation(info, (canonicalConfigFilePath, configFileName) => {
            let configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
            if (!configFileExistenceInfo) {
                // Create the cache
                configFileExistenceInfo = { exists: this.host.fileExists(configFileName) };
                this.configFileExistenceInfoCache.set(canonicalConfigFilePath, configFileExistenceInfo);
            }

            // Set this file as the root of inferred project
            (configFileExistenceInfo.openFilesImpactedByConfigFile ||= new ts.Map()).set(info.path, true);

            // If there is no configured project for this config file, add the file watcher
            configFileExistenceInfo.watcher ||= ts.canWatchDirectoryOrFile(ts.getDirectoryPath(canonicalConfigFilePath) as ts.Path) ?
                this.watchFactory.watchFile(
                    configFileName,
                    (_filename, eventKind) => this.onConfigFileChanged(canonicalConfigFilePath, eventKind),
                    ts.PollingInterval.High,
                    this.hostConfiguration.watchOptions,
                    ts.WatchType.ConfigFileForInferredRoot
                ) :
                noopConfigFileWatcher;
        });
    }

    /**
     * This is called by inferred project whenever root script info is removed from it
     */
    /* @internal */
    stopWatchingConfigFilesForInferredProjectRoot(info: ts.server.ScriptInfo) {
        this.forEachConfigFileLocation(info, canonicalConfigFilePath => {
            const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
            if (configFileExistenceInfo?.openFilesImpactedByConfigFile?.has(info.path)) {
                ts.Debug.assert(info.isScriptOpen());

                // Info is not root of inferred project any more
                configFileExistenceInfo.openFilesImpactedByConfigFile.set(info.path, false);

                // Close the config file watcher
                this.closeConfigFileWatcherOnReleaseOfOpenFile(configFileExistenceInfo);
            }
        });
    }

    /**
     * This function tries to search for a tsconfig.json for the given file.
     * This is different from the method the compiler uses because
     * the compiler can assume it will always start searching in the
     * current directory (the directory in which tsc was invoked).
     * The server must start searching from the directory containing
     * the newly opened file.
     */
    private forEachConfigFileLocation(info: OpenScriptInfoOrClosedOrConfigFileInfo, action: (canonicalConfigFilePath: ts.server.NormalizedPath, configFileName: ts.server.NormalizedPath) => boolean | void) {
        if (this.serverMode !== ts.LanguageServiceMode.Semantic) {
            return undefined;
        }

        ts.Debug.assert(!isOpenScriptInfo(info) || this.openFiles.has(info.path));
        const projectRootPath = this.openFiles.get(info.path);
        const scriptInfo = ts.Debug.checkDefined(this.getScriptInfo(info.path));
        if (scriptInfo.isDynamic) return undefined;

        let searchPath = ts.server.asNormalizedPath(ts.getDirectoryPath(info.fileName));
        const isSearchPathInProjectRoot = () => ts.containsPath(projectRootPath!, searchPath, this.currentDirectory, !this.host.useCaseSensitiveFileNames);

        // If projectRootPath doesn't contain info.path, then do normal search for config file
        const anySearchPathOk = !projectRootPath || !isSearchPathInProjectRoot();
        // For ancestor of config file always ignore its own directory since its going to result in itself
        let searchInDirectory = !isAncestorConfigFileInfo(info);
        do {
            if (searchInDirectory) {
                const canonicalSearchPath = ts.server.normalizedPathToPath(searchPath, this.currentDirectory, this.toCanonicalFileName);
                const tsconfigFileName = ts.server.asNormalizedPath(ts.combinePaths(searchPath, "tsconfig.json"));
                let result = action(ts.combinePaths(canonicalSearchPath, "tsconfig.json") as ts.server.NormalizedPath, tsconfigFileName);
                if (result) return tsconfigFileName;

                const jsconfigFileName = ts.server.asNormalizedPath(ts.combinePaths(searchPath, "jsconfig.json"));
                result = action(ts.combinePaths(canonicalSearchPath, "jsconfig.json") as ts.server.NormalizedPath, jsconfigFileName);
                if (result) return jsconfigFileName;

                // If we started within node_modules, don't look outside node_modules.
                // Otherwise, we might pick up a very large project and pull in the world,
                // causing an editor delay.
                if (ts.isNodeModulesDirectory(canonicalSearchPath)) {
                    break;
                }
            }

            const parentPath = ts.server.asNormalizedPath(ts.getDirectoryPath(searchPath));
            if (parentPath === searchPath) break;
            searchPath = parentPath;
            searchInDirectory = true;
        } while (anySearchPathOk || isSearchPathInProjectRoot());

        return undefined;
    }

    /*@internal*/
    findDefaultConfiguredProject(info: ts.server.ScriptInfo) {
        if (!info.isScriptOpen()) return undefined;
        const configFileName = this.getConfigFileNameForFile(info);
        const project = configFileName &&
            this.findConfiguredProjectByProjectName(configFileName);

        return project && projectContainsInfoDirectly(project, info) ?
            project :
            project?.getDefaultChildProjectFromProjectWithReferences(info);
    }

    /**
     * This function tries to search for a tsconfig.json for the given file.
     * This is different from the method the compiler uses because
     * the compiler can assume it will always start searching in the
     * current directory (the directory in which tsc was invoked).
     * The server must start searching from the directory containing
     * the newly opened file.
     * If script info is passed in, it is asserted to be open script info
     * otherwise just file name
     */
    private getConfigFileNameForFile(info: OpenScriptInfoOrClosedOrConfigFileInfo) {
        if (isOpenScriptInfo(info)) {
            ts.Debug.assert(info.isScriptOpen());
            const result = this.configFileForOpenFiles.get(info.path);
            if (result !== undefined) return result || undefined;
        }
        this.logger.info(`Search path: ${ts.getDirectoryPath(info.fileName)}`);
        const configFileName = this.forEachConfigFileLocation(info, (canonicalConfigFilePath, configFileName) =>
            this.configFileExists(configFileName, canonicalConfigFilePath, info));
        if (configFileName) {
            this.logger.info(`For info: ${info.fileName} :: Config file name: ${configFileName}`);
        }
        else {
            this.logger.info(`For info: ${info.fileName} :: No config files found.`);
        }
        if (isOpenScriptInfo(info)) {
            this.configFileForOpenFiles.set(info.path, configFileName || false);
        }
        return configFileName;
    }

    private printProjects() {
        if (!this.logger.hasLevel(ts.server.LogLevel.normal)) {
            return;
        }

        this.logger.startGroup();

        this.externalProjects.forEach(printProjectWithoutFileNames);
        this.configuredProjects.forEach(printProjectWithoutFileNames);
        this.inferredProjects.forEach(printProjectWithoutFileNames);

        this.logger.info("Open files: ");
        this.openFiles.forEach((projectRootPath, path) => {
            const info = this.getScriptInfoForPath(path as ts.Path)!;
            this.logger.info(`\tFileName: ${info.fileName} ProjectRootPath: ${projectRootPath}`);
            this.logger.info(`\t\tProjects: ${info.containingProjects.map(p => p.getProjectName())}`);
        });

        this.logger.endGroup();
    }

    /*@internal*/
    findConfiguredProjectByProjectName(configFileName: ts.server.NormalizedPath): ts.server.ConfiguredProject | undefined {
        // make sure that casing of config file name is consistent
        const canonicalConfigFilePath = ts.server.asNormalizedPath(this.toCanonicalFileName(configFileName));
        return this.getConfiguredProjectByCanonicalConfigFilePath(canonicalConfigFilePath);
    }

    private getConfiguredProjectByCanonicalConfigFilePath(canonicalConfigFilePath: string): ts.server.ConfiguredProject | undefined {
        return this.configuredProjects.get(canonicalConfigFilePath);
    }

    private findExternalProjectByProjectName(projectFileName: string) {
        return findProjectByName(projectFileName, this.externalProjects);
    }

    /** Get a filename if the language service exceeds the maximum allowed program size; otherwise returns undefined. */
    private getFilenameForExceededTotalSizeLimitForNonTsFiles<T>(name: string, options: ts.CompilerOptions | undefined, fileNames: T[], propertyReader: FilePropertyReader<T>): string | undefined {
        if (options && options.disableSizeLimit || !this.host.getFileSize) {
            return;
        }

        let availableSpace = maxProgramSizeForNonTsFiles;
        this.projectToSizeMap.set(name, 0);
        this.projectToSizeMap.forEach(val => (availableSpace -= (val || 0)));

        let totalNonTsFileSize = 0;

        for (const f of fileNames) {
            const fileName = propertyReader.getFileName(f);
            if (ts.hasTSFileExtension(fileName)) {
                continue;
            }

            totalNonTsFileSize += this.host.getFileSize(fileName);

            if (totalNonTsFileSize > maxProgramSizeForNonTsFiles || totalNonTsFileSize > availableSpace) {
                const top5LargestFiles = fileNames.map(f => propertyReader.getFileName(f))
                    .filter(name => !ts.hasTSFileExtension(name))
                    .map(name => ({ name, size: this.host.getFileSize!(name) }))
                    .sort((a, b) => b.size - a.size)
                    .slice(0, 5);
                this.logger.info(`Non TS file size exceeded limit (${totalNonTsFileSize}). Largest files: ${top5LargestFiles.map(file => `${file.name}:${file.size}`).join(", ")}`);
                // Keep the size as zero since it's disabled
                return fileName;
            }
        }
        this.projectToSizeMap.set(name, totalNonTsFileSize);
    }

    private createExternalProject(projectFileName: string, files: ts.server.protocol.ExternalFile[], options: ts.server.protocol.ExternalProjectCompilerOptions, typeAcquisition: ts.TypeAcquisition, excludedFiles: ts.server.NormalizedPath[]) {
        const compilerOptions = convertCompilerOptions(options);
        const watchOptionsAndErrors = convertWatchOptions(options, ts.getDirectoryPath(ts.normalizeSlashes(projectFileName)));
        const project = new ts.server.ExternalProject(
            projectFileName,
            this,
            this.documentRegistry,
            compilerOptions,
            /*lastFileExceededProgramSize*/ this.getFilenameForExceededTotalSizeLimitForNonTsFiles(projectFileName, compilerOptions, files, externalFilePropertyReader),
            options.compileOnSave === undefined ? true : options.compileOnSave,
            /*projectFilePath*/ undefined,
            this.currentPluginConfigOverrides,
            watchOptionsAndErrors?.watchOptions
        );
        project.setProjectErrors(watchOptionsAndErrors?.errors);
        project.excludedFiles = excludedFiles;

        this.addFilesToNonInferredProject(project, files, externalFilePropertyReader, typeAcquisition);
        this.externalProjects.push(project);
        return project;
    }

    /*@internal*/
    sendProjectTelemetry(project: ts.server.ExternalProject | ts.server.ConfiguredProject): void {
        if (this.seenProjects.has(project.projectName)) {
            setProjectOptionsUsed(project);
            return;
        }
        this.seenProjects.set(project.projectName, true);

        if (!this.eventHandler || !this.host.createSHA256Hash) {
            setProjectOptionsUsed(project);
            return;
        }

        const projectOptions = ts.server.isConfiguredProject(project) ? project.projectOptions as ts.server.ProjectOptions : undefined;
        setProjectOptionsUsed(project);
        const data: ProjectInfoTelemetryEventData = {
            projectId: this.host.createSHA256Hash(project.projectName),
            fileStats: ts.server.countEachFileTypes(project.getScriptInfos(), /*includeSizes*/ true),
            compilerOptions: ts.convertCompilerOptionsForTelemetry(project.getCompilationSettings()),
            typeAcquisition: convertTypeAcquisition(project.getTypeAcquisition()),
            extends: projectOptions && projectOptions.configHasExtendsProperty,
            files: projectOptions && projectOptions.configHasFilesProperty,
            include: projectOptions && projectOptions.configHasIncludeProperty,
            exclude: projectOptions && projectOptions.configHasExcludeProperty,
            compileOnSave: project.compileOnSaveEnabled,
            configFileName: configFileName(),
            projectType: project instanceof ts.server.ExternalProject ? "external" : "configured",
            languageServiceEnabled: project.languageServiceEnabled,
            version: ts.version, // eslint-disable-line @typescript-eslint/no-unnecessary-qualifier
        };
        this.eventHandler({ eventName: ProjectInfoTelemetryEvent, data });

        function configFileName(): ProjectInfoTelemetryEventData["configFileName"] {
            if (!ts.server.isConfiguredProject(project)) {
                return "other";
            }

            return ts.server.getBaseConfigFileName(project.getConfigFilePath()) || "other";
        }

        function convertTypeAcquisition({ enable, include, exclude }: ts.TypeAcquisition): ProjectInfoTypeAcquisitionData {
            return {
                enable,
                include: include !== undefined && include.length !== 0,
                exclude: exclude !== undefined && exclude.length !== 0,
            };
        }
    }

    private addFilesToNonInferredProject<T>(project: ts.server.ConfiguredProject | ts.server.ExternalProject, files: T[], propertyReader: FilePropertyReader<T>, typeAcquisition: ts.TypeAcquisition): void {
        this.updateNonInferredProjectFiles(project, files, propertyReader);
        project.setTypeAcquisition(typeAcquisition);
    }

    /* @internal */
    createConfiguredProject(configFileName: ts.server.NormalizedPath) {
        ts.tracing?.instant(ts.tracing.Phase.Session, "createConfiguredProject", { configFilePath: configFileName });
        this.logger.info(`Creating configuration project ${configFileName}`);
        const canonicalConfigFilePath = ts.server.asNormalizedPath(this.toCanonicalFileName(configFileName));
        let configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath);
        // We could be in this scenario if project is the configured project tracked by external project
        // Since that route doesnt check if the config file is present or not
        if (!configFileExistenceInfo) {
            this.configFileExistenceInfoCache.set(canonicalConfigFilePath, configFileExistenceInfo = { exists: true });
        }
        else {
            configFileExistenceInfo.exists = true;
        }
        if (!configFileExistenceInfo.config) {
            configFileExistenceInfo.config = {
                cachedDirectoryStructureHost: ts.createCachedDirectoryStructureHost(this.host, this.host.getCurrentDirectory(), this.host.useCaseSensitiveFileNames)!,
                projects: new ts.Map(),
                reloadLevel: ts.ConfigFileProgramReloadLevel.Full
            };
        }

        const project = new ts.server.ConfiguredProject(
            configFileName,
            canonicalConfigFilePath,
            this,
            this.documentRegistry,
            configFileExistenceInfo.config.cachedDirectoryStructureHost);
        this.configuredProjects.set(canonicalConfigFilePath, project);
        this.createConfigFileWatcherForParsedConfig(configFileName, canonicalConfigFilePath, project);
        return project;
    }

    /* @internal */
    private createConfiguredProjectWithDelayLoad(configFileName: ts.server.NormalizedPath, reason: string) {
        const project = this.createConfiguredProject(configFileName);
        project.pendingReload = ts.ConfigFileProgramReloadLevel.Full;
        project.pendingReloadReason = reason;
        return project;
    }

    /* @internal */
    createAndLoadConfiguredProject(configFileName: ts.server.NormalizedPath, reason: string) {
        const project = this.createConfiguredProject(configFileName);
        this.loadConfiguredProject(project, reason);
        return project;
    }

    /* @internal */
    private createLoadAndUpdateConfiguredProject(configFileName: ts.server.NormalizedPath, reason: string) {
        const project = this.createAndLoadConfiguredProject(configFileName, reason);
        project.updateGraph();
        return project;
    }

    /**
     * Read the config file of the project, and update the project root file names.
     */
    /* @internal */
    private loadConfiguredProject(project: ts.server.ConfiguredProject, reason: string) {
        ts.tracing?.push(ts.tracing.Phase.Session, "loadConfiguredProject", { configFilePath: project.canonicalConfigFilePath });
        this.sendProjectLoadingStartEvent(project, reason);

        // Read updated contents from disk
        const configFilename = ts.server.asNormalizedPath(ts.normalizePath(project.getConfigFilePath()));
        const configFileExistenceInfo = this.ensureParsedConfigUptoDate(
            configFilename,
            project.canonicalConfigFilePath,
            this.configFileExistenceInfoCache.get(project.canonicalConfigFilePath)!,
            project
        );
        const parsedCommandLine = configFileExistenceInfo.config!.parsedCommandLine!;
        ts.Debug.assert(!!parsedCommandLine.fileNames);
        const compilerOptions = parsedCommandLine.options;

        // Update the project
        if (!project.projectOptions) {
            project.projectOptions = {
                configHasExtendsProperty: parsedCommandLine.raw.extends !== undefined,
                configHasFilesProperty: parsedCommandLine.raw.files !== undefined,
                configHasIncludeProperty: parsedCommandLine.raw.include !== undefined,
                configHasExcludeProperty: parsedCommandLine.raw.exclude !== undefined
            };
        }
        project.canConfigFileJsonReportNoInputFiles = ts.canJsonReportNoInputFiles(parsedCommandLine.raw);
        project.setProjectErrors(parsedCommandLine.options.configFile!.parseDiagnostics);
        project.updateReferences(parsedCommandLine.projectReferences);
        const lastFileExceededProgramSize = this.getFilenameForExceededTotalSizeLimitForNonTsFiles(project.canonicalConfigFilePath, compilerOptions, parsedCommandLine.fileNames, fileNamePropertyReader);
        if (lastFileExceededProgramSize) {
            project.disableLanguageService(lastFileExceededProgramSize);
            this.configFileExistenceInfoCache.forEach((_configFileExistenceInfo, canonicalConfigFilePath) =>
                this.stopWatchingWildCards(canonicalConfigFilePath, project));
        }
        else {
            project.setCompilerOptions(compilerOptions);
            project.setWatchOptions(parsedCommandLine.watchOptions);
            project.enableLanguageService();
            this.watchWildcards(configFilename, configFileExistenceInfo, project);
        }
        project.enablePluginsWithOptions(compilerOptions, this.currentPluginConfigOverrides);
        const filesToAdd = parsedCommandLine.fileNames.concat(project.getExternalFiles());
        this.updateRootAndOptionsOfNonInferredProject(project, filesToAdd, fileNamePropertyReader, compilerOptions, parsedCommandLine.typeAcquisition!, parsedCommandLine.compileOnSave, parsedCommandLine.watchOptions);
        ts.tracing?.pop();
    }

    /*@internal*/
    ensureParsedConfigUptoDate(configFilename: ts.server.NormalizedPath, canonicalConfigFilePath: ts.server.NormalizedPath, configFileExistenceInfo: ConfigFileExistenceInfo, forProject: ts.server.ConfiguredProject): ConfigFileExistenceInfo {
        if (configFileExistenceInfo.config) {
            if (!configFileExistenceInfo.config.reloadLevel) return configFileExistenceInfo;
            if (configFileExistenceInfo.config.reloadLevel === ts.ConfigFileProgramReloadLevel.Partial) {
                this.reloadFileNamesOfParsedConfig(configFilename, configFileExistenceInfo.config);
                return configFileExistenceInfo;
            }
        }

        // Parse the config file and ensure its cached
        const cachedDirectoryStructureHost = configFileExistenceInfo.config?.cachedDirectoryStructureHost ||
            ts.createCachedDirectoryStructureHost(this.host, this.host.getCurrentDirectory(), this.host.useCaseSensitiveFileNames)!;

        // Read updated contents from disk
        const configFileContent = ts.tryReadFile(configFilename, fileName => this.host.readFile(fileName));
        const configFile = ts.parseJsonText(configFilename, ts.isString(configFileContent) ? configFileContent : "") as ts.TsConfigSourceFile;
        const configFileErrors = configFile.parseDiagnostics as ts.Diagnostic[];
        if (!ts.isString(configFileContent)) configFileErrors.push(configFileContent);
        const parsedCommandLine = ts.parseJsonSourceFileConfigFileContent(
            configFile,
            cachedDirectoryStructureHost,
            ts.getDirectoryPath(configFilename),
            /*existingOptions*/ {},
            configFilename,
            /*resolutionStack*/[],
            this.hostConfiguration.extraFileExtensions,
            this.extendedConfigCache,
        );

        if (parsedCommandLine.errors.length) {
            configFileErrors.push(...parsedCommandLine.errors);
        }

        this.logger.info(`Config: ${configFilename} : ${JSON.stringify({
            rootNames: parsedCommandLine.fileNames,
            options: parsedCommandLine.options,
            watchOptions: parsedCommandLine.watchOptions,
            projectReferences: parsedCommandLine.projectReferences
        }, /*replacer*/ undefined, " ")}`);

        const oldCommandLine = configFileExistenceInfo.config?.parsedCommandLine;
        if (!configFileExistenceInfo.config) {
            configFileExistenceInfo.config = { parsedCommandLine, cachedDirectoryStructureHost, projects: new ts.Map() };
        }
        else {
            configFileExistenceInfo.config.parsedCommandLine = parsedCommandLine;
            configFileExistenceInfo.config.watchedDirectoriesStale = true;
            configFileExistenceInfo.config.reloadLevel = undefined;
        }

        // If watch options different than older options when setting for the first time, update the config file watcher
        if (!oldCommandLine && !ts.isJsonEqual(
            // Old options
            this.getWatchOptionsFromProjectWatchOptions(/*projectOptions*/ undefined),
            // New options
            this.getWatchOptionsFromProjectWatchOptions(parsedCommandLine.watchOptions)
        )) {
            // Reset the config file watcher
            configFileExistenceInfo.watcher?.close();
            configFileExistenceInfo.watcher = undefined;
        }

        // Ensure there is watcher for this config file
        this.createConfigFileWatcherForParsedConfig(configFilename, canonicalConfigFilePath, forProject);
        // Watch extended config files
        ts.updateSharedExtendedConfigFileWatcher(
            canonicalConfigFilePath,
            parsedCommandLine.options,
            this.sharedExtendedConfigFileWatchers,
            (extendedConfigFileName, extendedConfigFilePath) => this.watchFactory.watchFile(
                extendedConfigFileName,
                () => {
                    // Update extended config cache
                    ts.cleanExtendedConfigCache(this.extendedConfigCache, extendedConfigFilePath, fileName => this.toPath(fileName));
                    // Update projects
                    let ensureProjectsForOpenFiles = false;
                    this.sharedExtendedConfigFileWatchers.get(extendedConfigFilePath)?.projects.forEach(canonicalPath => {
                        ensureProjectsForOpenFiles = this.delayUpdateProjectsFromParsedConfigOnConfigFileChange(canonicalPath, `Change in extended config file ${extendedConfigFileName} detected`) || ensureProjectsForOpenFiles;
                    });
                    if (ensureProjectsForOpenFiles) this.delayEnsureProjectForOpenFiles();
                },
                ts.PollingInterval.High,
                this.hostConfiguration.watchOptions,
                ts.WatchType.ExtendedConfigFile,
                configFilename
            ),
            fileName => this.toPath(fileName),
        );
        return configFileExistenceInfo;
    }

    /*@internal*/
    watchWildcards(configFileName: ts.server.NormalizedPath, { exists, config }: ConfigFileExistenceInfo, forProject: ts.server.ConfiguredProject) {
        config!.projects.set(forProject.canonicalConfigFilePath, true);
        if (exists) {
            if (config!.watchedDirectories && !config!.watchedDirectoriesStale) return;
            config!.watchedDirectoriesStale = false;
            ts.updateWatchingWildcardDirectories(
                config!.watchedDirectories ||= new ts.Map(),
                new ts.Map(ts.getEntries(config!.parsedCommandLine!.wildcardDirectories!)),
                // Create new directory watcher
                (directory, flags) => this.watchWildcardDirectory(directory as ts.Path, flags, configFileName, config!),
            );
        }
        else {
            config!.watchedDirectoriesStale = false;
            if (!config!.watchedDirectories) return;
            ts.clearMap(config!.watchedDirectories, ts.closeFileWatcherOf);
            config!.watchedDirectories = undefined;
        }
    }

    /*@internal*/
    stopWatchingWildCards(canonicalConfigFilePath: ts.server.NormalizedPath, forProject: ts.server.ConfiguredProject) {
        const configFileExistenceInfo = this.configFileExistenceInfoCache.get(canonicalConfigFilePath)!;
        if (!configFileExistenceInfo.config ||
            !configFileExistenceInfo.config.projects.get(forProject.canonicalConfigFilePath)) {
            return;
        }

        configFileExistenceInfo.config.projects.set(forProject.canonicalConfigFilePath, false);
        // If any of the project is still watching wild cards dont close the watcher
        if (ts.forEachEntry(configFileExistenceInfo.config.projects, ts.identity)) return;

        if (configFileExistenceInfo.config.watchedDirectories) {
            ts.clearMap(configFileExistenceInfo.config.watchedDirectories, ts.closeFileWatcherOf);
            configFileExistenceInfo.config.watchedDirectories = undefined;
        }
        configFileExistenceInfo.config.watchedDirectoriesStale = undefined;
    }

    private updateNonInferredProjectFiles<T>(project: ts.server.Project, files: T[], propertyReader: FilePropertyReader<T>) {
        const projectRootFilesMap = project.getRootFilesMap();
        const newRootScriptInfoMap = new ts.Map<string, true>();

        for (const f of files) {
            const newRootFile = propertyReader.getFileName(f);
            const fileName = ts.server.toNormalizedPath(newRootFile);
            const isDynamic = ts.server.isDynamicFileName(fileName);
            let path: ts.Path;
            // Use the project's fileExists so that it can use caching instead of reaching to disk for the query
            if (!isDynamic && !project.fileExists(newRootFile)) {
                path = ts.server.normalizedPathToPath(fileName, this.currentDirectory, this.toCanonicalFileName);
                const existingValue = projectRootFilesMap.get(path);
                if (existingValue) {
                    if (existingValue.info) {
                        project.removeFile(existingValue.info, /*fileExists*/ false, /*detachFromProject*/ true);
                        existingValue.info = undefined;
                    }
                    existingValue.fileName = fileName;
                }
                else {
                    projectRootFilesMap.set(path, { fileName });
                }
            }
            else {
                const scriptKind = propertyReader.getScriptKind(f, this.hostConfiguration.extraFileExtensions);
                const hasMixedContent = propertyReader.hasMixedContent(f, this.hostConfiguration.extraFileExtensions);
                const scriptInfo = ts.Debug.checkDefined(this.getOrCreateScriptInfoNotOpenedByClientForNormalizedPath(
                    fileName,
                    project.currentDirectory,
                    scriptKind,
                    hasMixedContent,
                    project.directoryStructureHost
                ));
                path = scriptInfo.path;
                const existingValue = projectRootFilesMap.get(path);
                // If this script info is not already a root add it
                if (!existingValue || existingValue.info !== scriptInfo) {
                    project.addRoot(scriptInfo, fileName);
                    if (scriptInfo.isScriptOpen()) {
                        // if file is already root in some inferred project
                        // - remove the file from that project and delete the project if necessary
                        this.removeRootOfInferredProjectIfNowPartOfOtherProject(scriptInfo);
                    }
                }
                else {
                    // Already root update the fileName
                    existingValue.fileName = fileName;
                }
            }
            newRootScriptInfoMap.set(path, true);
        }

        // project's root file map size is always going to be same or larger than new roots map
        // as we have already all the new files to the project
        if (projectRootFilesMap.size > newRootScriptInfoMap.size) {
            projectRootFilesMap.forEach((value, path) => {
                if (!newRootScriptInfoMap.has(path)) {
                    if (value.info) {
                        project.removeFile(value.info, project.fileExists(path), /*detachFromProject*/ true);
                    }
                    else {
                        projectRootFilesMap.delete(path);
                    }
                }
            });
        }

        // Just to ensure that even if root files dont change, the changes to the non root file are picked up,
        // mark the project as dirty unconditionally
        project.markAsDirty();
    }

    private updateRootAndOptionsOfNonInferredProject<T>(project: ts.server.ExternalProject | ts.server.ConfiguredProject, newUncheckedFiles: T[], propertyReader: FilePropertyReader<T>, newOptions: ts.CompilerOptions, newTypeAcquisition: ts.TypeAcquisition, compileOnSave: boolean | undefined, watchOptions: ts.WatchOptions | undefined) {
        project.setCompilerOptions(newOptions);
        project.setWatchOptions(watchOptions);
        // VS only set the CompileOnSaveEnabled option in the request if the option was changed recently
        // therefore if it is undefined, it should not be updated.
        if (compileOnSave !== undefined) {
            project.compileOnSaveEnabled = compileOnSave;
        }
        this.addFilesToNonInferredProject(project, newUncheckedFiles, propertyReader, newTypeAcquisition);
    }

    /**
     * Reload the file names from config file specs and update the project graph
     */
    /*@internal*/
    reloadFileNamesOfConfiguredProject(project: ts.server.ConfiguredProject) {
        const fileNames = this.reloadFileNamesOfParsedConfig(project.getConfigFilePath(), this.configFileExistenceInfoCache.get(project.canonicalConfigFilePath)!.config!);
        project.updateErrorOnNoInputFiles(fileNames);
        this.updateNonInferredProjectFiles(project, fileNames.concat(project.getExternalFiles()), fileNamePropertyReader);
        return project.updateGraph();
    }

    /*@internal*/
    private reloadFileNamesOfParsedConfig(configFileName: ts.server.NormalizedPath, config: ParsedConfig) {
        if (config.reloadLevel === undefined) return config.parsedCommandLine!.fileNames;
        ts.Debug.assert(config.reloadLevel === ts.ConfigFileProgramReloadLevel.Partial);
        const configFileSpecs = config.parsedCommandLine!.options.configFile!.configFileSpecs!;
        const fileNames = ts.getFileNamesFromConfigSpecs(
            configFileSpecs,
            ts.getDirectoryPath(configFileName),
            config.parsedCommandLine!.options,
            config.cachedDirectoryStructureHost,
            this.hostConfiguration.extraFileExtensions
        );
        config.parsedCommandLine = { ...config.parsedCommandLine!, fileNames };
        return fileNames;
    }

    /*@internal*/
    setFileNamesOfAutoImportProviderProject(project: ts.server.AutoImportProviderProject, fileNames: string[]) {
        this.updateNonInferredProjectFiles(project, fileNames, fileNamePropertyReader);
    }

    /**
     * Read the config file of the project again by clearing the cache and update the project graph
     */
    /* @internal */
    reloadConfiguredProject(project: ts.server.ConfiguredProject, reason: string, isInitialLoad: boolean, clearSemanticCache: boolean) {
        // At this point, there is no reason to not have configFile in the host
        const host = project.getCachedDirectoryStructureHost();
        if (clearSemanticCache) this.clearSemanticCache(project);

        // Clear the cache since we are reloading the project from disk
        host.clearCache();
        const configFileName = project.getConfigFilePath();
        this.logger.info(`${isInitialLoad ? "Loading" : "Reloading"} configured project ${configFileName}`);

        // Load project from the disk
        this.loadConfiguredProject(project, reason);
        project.updateGraph();

        this.sendConfigFileDiagEvent(project, configFileName);
    }

    /* @internal */
    private clearSemanticCache(project: ts.server.Project) {
        project.resolutionCache.clear();
        project.getLanguageService(/*ensureSynchronized*/ false).cleanupSemanticCache();
        project.markAsDirty();
    }

    private sendConfigFileDiagEvent(project: ts.server.ConfiguredProject, triggerFile: ts.server.NormalizedPath) {
        if (!this.eventHandler || this.suppressDiagnosticEvents) {
            return;
        }
        const diagnostics = project.getLanguageService().getCompilerOptionsDiagnostics();
        diagnostics.push(...project.getAllProjectErrors());

        this.eventHandler({
            eventName: ConfigFileDiagEvent,
            data: { configFileName: project.getConfigFilePath(), diagnostics, triggerFile }
        } as ConfigFileDiagEvent);
    }

    private getOrCreateInferredProjectForProjectRootPathIfEnabled(info: ts.server.ScriptInfo, projectRootPath: ts.server.NormalizedPath | undefined): ts.server.InferredProject | undefined {
        if (!this.useInferredProjectPerProjectRoot ||
            // Its a dynamic info opened without project root
            (info.isDynamic && projectRootPath === undefined)) {
            return undefined;
        }

        if (projectRootPath) {
            const canonicalProjectRootPath = this.toCanonicalFileName(projectRootPath);
            // if we have an explicit project root path, find (or create) the matching inferred project.
            for (const project of this.inferredProjects) {
                if (project.projectRootPath === canonicalProjectRootPath) {
                    return project;
                }
            }
            return this.createInferredProject(projectRootPath, /*isSingleInferredProject*/ false, projectRootPath);
        }

        // we don't have an explicit root path, so we should try to find an inferred project
        // that more closely contains the file.
        let bestMatch: ts.server.InferredProject | undefined;
        for (const project of this.inferredProjects) {
            // ignore single inferred projects (handled elsewhere)
            if (!project.projectRootPath) continue;
            // ignore inferred projects that don't contain the root's path
            if (!ts.containsPath(project.projectRootPath, info.path, this.host.getCurrentDirectory(), !this.host.useCaseSensitiveFileNames)) continue;
            // ignore inferred projects that are higher up in the project root.
            // TODO(rbuckton): Should we add the file as a root to these as well?
            if (bestMatch && bestMatch.projectRootPath!.length > project.projectRootPath.length) continue;
            bestMatch = project;
        }

        return bestMatch;
    }

    private getOrCreateSingleInferredProjectIfEnabled(): ts.server.InferredProject | undefined {
        if (!this.useSingleInferredProject) {
            return undefined;
        }

        // If `useInferredProjectPerProjectRoot` is not enabled, then there will only be one
        // inferred project for all files. If `useInferredProjectPerProjectRoot` is enabled
        // then we want to put all files that are not opened with a `projectRootPath` into
        // the same inferred project.
        //
        // To avoid the cost of searching through the array and to optimize for the case where
        // `useInferredProjectPerProjectRoot` is not enabled, we will always put the inferred
        // project for non-rooted files at the front of the array.
        if (this.inferredProjects.length > 0 && this.inferredProjects[0].projectRootPath === undefined) {
            return this.inferredProjects[0];
        }

        // Single inferred project does not have a project root and hence no current directory
        return this.createInferredProject(/*currentDirectory*/ undefined, /*isSingleInferredProject*/ true);
    }

    private getOrCreateSingleInferredWithoutProjectRoot(currentDirectory: string | undefined): ts.server.InferredProject {
        ts.Debug.assert(!this.useSingleInferredProject);
        const expectedCurrentDirectory = this.toCanonicalFileName(this.getNormalizedAbsolutePath(currentDirectory || ""));
        // Reuse the project with same current directory but no roots
        for (const inferredProject of this.inferredProjects) {
            if (!inferredProject.projectRootPath &&
                inferredProject.isOrphan() &&
                inferredProject.canonicalCurrentDirectory === expectedCurrentDirectory) {
                return inferredProject;
            }
        }

        return this.createInferredProject(currentDirectory);
    }

    private createInferredProject(currentDirectory: string | undefined, isSingleInferredProject?: boolean, projectRootPath?: ts.server.NormalizedPath): ts.server.InferredProject {
        const compilerOptions = projectRootPath && this.compilerOptionsForInferredProjectsPerProjectRoot.get(projectRootPath) || this.compilerOptionsForInferredProjects!; // TODO: GH#18217
        let watchOptionsAndErrors: WatchOptionsAndErrors | false | undefined;
        let typeAcquisition: ts.TypeAcquisition | undefined;
        if (projectRootPath) {
            watchOptionsAndErrors = this.watchOptionsForInferredProjectsPerProjectRoot.get(projectRootPath);
            typeAcquisition = this.typeAcquisitionForInferredProjectsPerProjectRoot.get(projectRootPath);
        }
        if (watchOptionsAndErrors === undefined) {
            watchOptionsAndErrors = this.watchOptionsForInferredProjects;
        }
        if (typeAcquisition === undefined) {
            typeAcquisition = this.typeAcquisitionForInferredProjects;
        }
        watchOptionsAndErrors = watchOptionsAndErrors || undefined;
        const project = new ts.server.InferredProject(this, this.documentRegistry, compilerOptions, watchOptionsAndErrors?.watchOptions, projectRootPath, currentDirectory, this.currentPluginConfigOverrides, typeAcquisition);
        project.setProjectErrors(watchOptionsAndErrors?.errors);
        if (isSingleInferredProject) {
            this.inferredProjects.unshift(project);
        }
        else {
            this.inferredProjects.push(project);
        }
        return project;
    }

    /*@internal*/
    getOrCreateScriptInfoNotOpenedByClient(uncheckedFileName: string, currentDirectory: string, hostToQueryFileExistsOn: ts.DirectoryStructureHost) {
        return this.getOrCreateScriptInfoNotOpenedByClientForNormalizedPath(
            ts.server.toNormalizedPath(uncheckedFileName), currentDirectory, /*scriptKind*/ undefined,
            /*hasMixedContent*/ undefined, hostToQueryFileExistsOn
        );
    }

    getScriptInfo(uncheckedFileName: string) {
        return this.getScriptInfoForNormalizedPath(ts.server.toNormalizedPath(uncheckedFileName));
    }

    /* @internal */
    getScriptInfoOrConfig(uncheckedFileName: string): ScriptInfoOrConfig | undefined {
        const path = ts.server.toNormalizedPath(uncheckedFileName);
        const info = this.getScriptInfoForNormalizedPath(path);
        if (info) return info;
        const configProject = this.configuredProjects.get(this.toPath(uncheckedFileName));
        return configProject && configProject.getCompilerOptions().configFile;
    }

    /* @internal */
    logErrorForScriptInfoNotFound(fileName: string): void {
        const names = ts.arrayFrom(this.filenameToScriptInfo.entries()).map(([path, scriptInfo]) => ({ path, fileName: scriptInfo.fileName }));
        this.logger.msg(`Could not find file ${JSON.stringify(fileName)}.\nAll files are: ${JSON.stringify(names)}`, ts.server.Msg.Err);
    }

    /**
     * Returns the projects that contain script info through SymLink
     * Note that this does not return projects in info.containingProjects
     */
    /*@internal*/
    getSymlinkedProjects(info: ts.server.ScriptInfo): ts.MultiMap<ts.Path, ts.server.Project> | undefined {
        let projects: ts.MultiMap<ts.Path, ts.server.Project> | undefined;
        if (this.realpathToScriptInfos) {
            const realpath = info.getRealpathIfDifferent();
            if (realpath) {
                ts.forEach(this.realpathToScriptInfos.get(realpath), combineProjects);
            }
            ts.forEach(this.realpathToScriptInfos.get(info.path), combineProjects);
        }

        return projects;

        function combineProjects(toAddInfo: ts.server.ScriptInfo) {
            if (toAddInfo !== info) {
                for (const project of toAddInfo.containingProjects) {
                    // Add the projects only if they can use symLink targets and not already in the list
                    if (project.languageServiceEnabled &&
                        !project.isOrphan() &&
                        !project.getCompilerOptions().preserveSymlinks &&
                        !info.isAttached(project)) {
                        if (!projects) {
                            projects = ts.createMultiMap();
                            projects.add(toAddInfo.path, project);
                        }
                        else if (!ts.forEachEntry(projects, (projs, path) => path === toAddInfo.path ? false : ts.contains(projs, project))) {
                            projects.add(toAddInfo.path, project);
                        }
                    }
                }
            }
        }
    }

    private watchClosedScriptInfo(info: ts.server.ScriptInfo) {
        ts.Debug.assert(!info.fileWatcher);
        // do not watch files with mixed content - server doesn't know how to interpret it
        // do not watch files in the global cache location
        if (!info.isDynamicOrHasMixedContent() &&
            (!this.globalCacheLocationDirectoryPath ||
                !ts.startsWith(info.path, this.globalCacheLocationDirectoryPath))) {
            const indexOfNodeModules = info.path.indexOf("/node_modules/");
            if (!this.host.getModifiedTime || indexOfNodeModules === -1) {
                info.fileWatcher = this.watchFactory.watchFile(
                    info.fileName,
                    (_fileName, eventKind) => this.onSourceFileChanged(info, eventKind),
                    ts.PollingInterval.Medium,
                    this.hostConfiguration.watchOptions,
                    ts.WatchType.ClosedScriptInfo
                );
            }
            else {
                info.mTime = this.getModifiedTime(info);
                info.fileWatcher = this.watchClosedScriptInfoInNodeModules(info.path.substr(0, indexOfNodeModules) as ts.Path);
            }
        }
    }

    private createNodeModulesWatcher(dir: ts.Path) {
        const watcher = this.watchFactory.watchDirectory(
            dir,
            fileOrDirectory => {
                const fileOrDirectoryPath = ts.removeIgnoredPath(this.toPath(fileOrDirectory));
                if (!fileOrDirectoryPath) return;

                // Clear module specifier cache for any projects whose cache was affected by
                // dependency package.jsons in this node_modules directory
                const basename = ts.getBaseFileName(fileOrDirectoryPath);
                if (result.affectedModuleSpecifierCacheProjects?.size && (
                    basename === "package.json" || basename === "node_modules"
                )) {
                    result.affectedModuleSpecifierCacheProjects.forEach(projectName => {
                        this.findProject(projectName)?.getModuleSpecifierCache()?.clear();
                    });
                }

                // Refresh closed script info after an npm install
                if (result.refreshScriptInfoRefCount) {
                    if (dir === fileOrDirectoryPath) {
                        this.refreshScriptInfosInDirectory(dir);
                    }
                    else {
                        const info = this.getScriptInfoForPath(fileOrDirectoryPath);
                        if (info) {
                            if (isScriptInfoWatchedFromNodeModules(info)) {
                                this.refreshScriptInfo(info);
                            }
                        }
                        // Folder
                        else if (!ts.hasExtension(fileOrDirectoryPath)) {
                            this.refreshScriptInfosInDirectory(fileOrDirectoryPath);
                        }
                    }
                }
            },
            ts.WatchDirectoryFlags.Recursive,
            this.hostConfiguration.watchOptions,
            ts.WatchType.NodeModules
        );
        const result: NodeModulesWatcher = {
            refreshScriptInfoRefCount: 0,
            affectedModuleSpecifierCacheProjects: undefined,
            close: () => {
                if (!result.refreshScriptInfoRefCount && !result.affectedModuleSpecifierCacheProjects?.size) {
                    watcher.close();
                    this.nodeModulesWatchers.delete(dir);
                }
            },
        };
        this.nodeModulesWatchers.set(dir, result);
        return result;
    }

    /*@internal*/
    watchPackageJsonsInNodeModules(dir: ts.Path, project: ts.server.Project): ts.FileWatcher {
        const watcher = this.nodeModulesWatchers.get(dir) || this.createNodeModulesWatcher(dir);
        (watcher.affectedModuleSpecifierCacheProjects ||= new ts.Set()).add(project.getProjectName());

        return {
            close: () => {
                watcher.affectedModuleSpecifierCacheProjects?.delete(project.getProjectName());
                watcher.close();
            },
        };
    }

    private watchClosedScriptInfoInNodeModules(dir: ts.Path): ts.FileWatcher {
        const watchDir = dir + "/node_modules" as ts.Path;
        const watcher = this.nodeModulesWatchers.get(watchDir) || this.createNodeModulesWatcher(watchDir);
        watcher.refreshScriptInfoRefCount++;

        return {
            close: () => {
                watcher.refreshScriptInfoRefCount--;
                watcher.close();
            },
        };
    }

    private getModifiedTime(info: ts.server.ScriptInfo) {
        return (this.host.getModifiedTime!(info.path) || ts.missingFileModifiedTime).getTime();
    }

    private refreshScriptInfo(info: ts.server.ScriptInfo) {
        const mTime = this.getModifiedTime(info);
        if (mTime !== info.mTime) {
            const eventKind = ts.getFileWatcherEventKind(info.mTime!, mTime);
            info.mTime = mTime;
            this.onSourceFileChanged(info, eventKind);
        }
    }

    private refreshScriptInfosInDirectory(dir: ts.Path) {
        dir = dir + ts.directorySeparator as ts.Path;
        this.filenameToScriptInfo.forEach(info => {
            if (isScriptInfoWatchedFromNodeModules(info) && ts.startsWith(info.path, dir)) {
                this.refreshScriptInfo(info);
            }
        });
    }

    private stopWatchingScriptInfo(info: ts.server.ScriptInfo) {
        if (info.fileWatcher) {
            info.fileWatcher.close();
            info.fileWatcher = undefined;
        }
    }

    private getOrCreateScriptInfoNotOpenedByClientForNormalizedPath(fileName: ts.server.NormalizedPath, currentDirectory: string, scriptKind: ts.ScriptKind | undefined, hasMixedContent: boolean | undefined, hostToQueryFileExistsOn: ts.DirectoryStructureHost | undefined) {
        if (ts.isRootedDiskPath(fileName) || ts.server.isDynamicFileName(fileName)) {
            return this.getOrCreateScriptInfoWorker(fileName, currentDirectory, /*openedByClient*/ false, /*fileContent*/ undefined, scriptKind, hasMixedContent, hostToQueryFileExistsOn);
        }

        // This is non rooted path with different current directory than project service current directory
        // Only paths recognized are open relative file paths
        const info = this.openFilesWithNonRootedDiskPath.get(this.toCanonicalFileName(fileName));
        if (info) {
            return info;
        }

        // This means triple slash references wont be resolved in dynamic and unsaved files
        // which is intentional since we dont know what it means to be relative to non disk files
        return undefined;
    }

    private getOrCreateScriptInfoOpenedByClientForNormalizedPath(fileName: ts.server.NormalizedPath, currentDirectory: string, fileContent: string | undefined, scriptKind: ts.ScriptKind | undefined, hasMixedContent: boolean | undefined) {
        return this.getOrCreateScriptInfoWorker(fileName, currentDirectory, /*openedByClient*/ true, fileContent, scriptKind, hasMixedContent);
    }

    getOrCreateScriptInfoForNormalizedPath(fileName: ts.server.NormalizedPath, openedByClient: boolean, fileContent?: string, scriptKind?: ts.ScriptKind, hasMixedContent?: boolean, hostToQueryFileExistsOn?: { fileExists(path: string): boolean; }) {
        return this.getOrCreateScriptInfoWorker(fileName, this.currentDirectory, openedByClient, fileContent, scriptKind, hasMixedContent, hostToQueryFileExistsOn);
    }

    private getOrCreateScriptInfoWorker(fileName: ts.server.NormalizedPath, currentDirectory: string, openedByClient: boolean, fileContent?: string, scriptKind?: ts.ScriptKind, hasMixedContent?: boolean, hostToQueryFileExistsOn?: { fileExists(path: string): boolean; }) {
        ts.Debug.assert(fileContent === undefined || openedByClient, "ScriptInfo needs to be opened by client to be able to set its user defined content");
        const path = ts.server.normalizedPathToPath(fileName, currentDirectory, this.toCanonicalFileName);
        let info = this.getScriptInfoForPath(path);
        if (!info) {
            const isDynamic = ts.server.isDynamicFileName(fileName);
            ts.Debug.assert(ts.isRootedDiskPath(fileName) || isDynamic || openedByClient, "", () => `${JSON.stringify({ fileName, currentDirectory, hostCurrentDirectory: this.currentDirectory, openKeys: ts.arrayFrom(this.openFilesWithNonRootedDiskPath.keys()) })}\nScript info with non-dynamic relative file name can only be open script info or in context of host currentDirectory`);
            ts.Debug.assert(!ts.isRootedDiskPath(fileName) || this.currentDirectory === currentDirectory || !this.openFilesWithNonRootedDiskPath.has(this.toCanonicalFileName(fileName)), "", () => `${JSON.stringify({ fileName, currentDirectory, hostCurrentDirectory: this.currentDirectory, openKeys: ts.arrayFrom(this.openFilesWithNonRootedDiskPath.keys()) })}\nOpen script files with non rooted disk path opened with current directory context cannot have same canonical names`);
            ts.Debug.assert(!isDynamic || this.currentDirectory === currentDirectory || this.useInferredProjectPerProjectRoot, "", () => `${JSON.stringify({ fileName, currentDirectory, hostCurrentDirectory: this.currentDirectory, openKeys: ts.arrayFrom(this.openFilesWithNonRootedDiskPath.keys()) })}\nDynamic files must always be opened with service's current directory or service should support inferred project per projectRootPath.`);
            // If the file is not opened by client and the file doesnot exist on the disk, return
            if (!openedByClient && !isDynamic && !(hostToQueryFileExistsOn || this.host).fileExists(fileName)) {
                return;
            }
            info = new ts.server.ScriptInfo(this.host, fileName, scriptKind!, !!hasMixedContent, path, this.filenameToScriptInfoVersion.get(path)); // TODO: GH#18217
            this.filenameToScriptInfo.set(info.path, info);
            this.filenameToScriptInfoVersion.delete(info.path);
            if (!openedByClient) {
                this.watchClosedScriptInfo(info);
            }
            else if (!ts.isRootedDiskPath(fileName) && (!isDynamic || this.currentDirectory !== currentDirectory)) {
                // File that is opened by user but isn't rooted disk path
                this.openFilesWithNonRootedDiskPath.set(this.toCanonicalFileName(fileName), info);
            }
        }
        if (openedByClient) {
            // Opening closed script info
            // either it was created just now, or was part of projects but was closed
            this.stopWatchingScriptInfo(info);
            info.open(fileContent!);
            if (hasMixedContent) {
                info.registerFileUpdate();
            }
        }
        return info;
    }

    /**
     * This gets the script info for the normalized path. If the path is not rooted disk path then the open script info with project root context is preferred
     */
    getScriptInfoForNormalizedPath(fileName: ts.server.NormalizedPath) {
        return !ts.isRootedDiskPath(fileName) && this.openFilesWithNonRootedDiskPath.get(this.toCanonicalFileName(fileName)) ||
            this.getScriptInfoForPath(ts.server.normalizedPathToPath(fileName, this.currentDirectory, this.toCanonicalFileName));
    }

    getScriptInfoForPath(fileName: ts.Path) {
        return this.filenameToScriptInfo.get(fileName);
    }

    /*@internal*/
    getDocumentPositionMapper(project: ts.server.Project, generatedFileName: string, sourceFileName?: string): ts.DocumentPositionMapper | undefined {
        // Since declaration info and map file watches arent updating project's directory structure host (which can cache file structure) use host
        const declarationInfo = this.getOrCreateScriptInfoNotOpenedByClient(generatedFileName, project.currentDirectory, this.host);
        if (!declarationInfo) {
            if (sourceFileName) {
                // Project contains source file and it generates the generated file name
                project.addGeneratedFileWatch(generatedFileName, sourceFileName);
            }
            return undefined;
        }

        // Try to get from cache
        declarationInfo.getSnapshot(); // Ensure synchronized
        if (ts.isString(declarationInfo.sourceMapFilePath)) {
            // Ensure mapper is synchronized
            const sourceMapFileInfo = this.getScriptInfoForPath(declarationInfo.sourceMapFilePath);
            if (sourceMapFileInfo) {
                sourceMapFileInfo.getSnapshot();
                if (sourceMapFileInfo.documentPositionMapper !== undefined) {
                    sourceMapFileInfo.sourceInfos = this.addSourceInfoToSourceMap(sourceFileName, project, sourceMapFileInfo.sourceInfos);
                    return sourceMapFileInfo.documentPositionMapper ? sourceMapFileInfo.documentPositionMapper : undefined;
                }
            }
            declarationInfo.sourceMapFilePath = undefined;
        }
        else if (declarationInfo.sourceMapFilePath) {
            declarationInfo.sourceMapFilePath.sourceInfos = this.addSourceInfoToSourceMap(sourceFileName, project, declarationInfo.sourceMapFilePath.sourceInfos);
            return undefined;
        }
        else if (declarationInfo.sourceMapFilePath !== undefined) {
            // Doesnt have sourceMap
            return undefined;
        }

        // Create the mapper
        let sourceMapFileInfo: ts.server.ScriptInfo | undefined;
        let mapFileNameFromDeclarationInfo: string | undefined;

        let readMapFile: ts.ReadMapFile | undefined = (mapFileName, mapFileNameFromDts) => {
            const mapInfo = this.getOrCreateScriptInfoNotOpenedByClient(mapFileName, project.currentDirectory, this.host);
            if (!mapInfo) {
                mapFileNameFromDeclarationInfo = mapFileNameFromDts;
                return undefined;
            }
            sourceMapFileInfo = mapInfo;
            const snap = mapInfo.getSnapshot();
            if (mapInfo.documentPositionMapper !== undefined) return mapInfo.documentPositionMapper;
            return ts.getSnapshotText(snap);
        };
        const projectName = project.projectName;
        const documentPositionMapper = ts.getDocumentPositionMapper(
            { getCanonicalFileName: this.toCanonicalFileName, log: s => this.logger.info(s), getSourceFileLike: f => this.getSourceFileLike(f, projectName, declarationInfo) },
            declarationInfo.fileName,
            declarationInfo.getLineInfo(),
            readMapFile
        );
        readMapFile = undefined; // Remove ref to project
        if (sourceMapFileInfo) {
            declarationInfo.sourceMapFilePath = sourceMapFileInfo.path;
            sourceMapFileInfo.declarationInfoPath = declarationInfo.path;
            sourceMapFileInfo.documentPositionMapper = documentPositionMapper || false;
            sourceMapFileInfo.sourceInfos = this.addSourceInfoToSourceMap(sourceFileName, project, sourceMapFileInfo.sourceInfos);
        }
        else if (mapFileNameFromDeclarationInfo) {
            declarationInfo.sourceMapFilePath = {
                watcher: this.addMissingSourceMapFile(
                    project.currentDirectory === this.currentDirectory ?
                        mapFileNameFromDeclarationInfo :
                        ts.getNormalizedAbsolutePath(mapFileNameFromDeclarationInfo, project.currentDirectory),
                    declarationInfo.path
                ),
                sourceInfos: this.addSourceInfoToSourceMap(sourceFileName, project)
            };
        }
        else {
            declarationInfo.sourceMapFilePath = false;
        }
        return documentPositionMapper;
    }

    private addSourceInfoToSourceMap(sourceFileName: string | undefined, project: ts.server.Project, sourceInfos?: ts.Set<ts.Path>) {
        if (sourceFileName) {
            // Attach as source
            const sourceInfo = this.getOrCreateScriptInfoNotOpenedByClient(sourceFileName, project.currentDirectory, project.directoryStructureHost)!;
            (sourceInfos || (sourceInfos = new ts.Set())).add(sourceInfo.path);
        }
        return sourceInfos;
    }

    private addMissingSourceMapFile(mapFileName: string, declarationInfoPath: ts.Path) {
        const fileWatcher = this.watchFactory.watchFile(
            mapFileName,
            () => {
                const declarationInfo = this.getScriptInfoForPath(declarationInfoPath);
                if (declarationInfo && declarationInfo.sourceMapFilePath && !ts.isString(declarationInfo.sourceMapFilePath)) {
                    // Update declaration and source projects
                    this.delayUpdateProjectGraphs(declarationInfo.containingProjects, /*clearSourceMapperCache*/ true);
                    this.delayUpdateSourceInfoProjects(declarationInfo.sourceMapFilePath.sourceInfos);
                    declarationInfo.closeSourceMapFileWatcher();
                }
            },
            ts.PollingInterval.High,
            this.hostConfiguration.watchOptions,
            ts.WatchType.MissingSourceMapFile,
        );
        return fileWatcher;
    }

    /*@internal*/
    getSourceFileLike(fileName: string, projectNameOrProject: string | ts.server.Project, declarationInfo?: ts.server.ScriptInfo) {
        const project = (projectNameOrProject as ts.server.Project).projectName ? projectNameOrProject as ts.server.Project : this.findProject(projectNameOrProject as string);
        if (project) {
            const path = project.toPath(fileName);
            const sourceFile = project.getSourceFile(path);
            if (sourceFile && sourceFile.resolvedPath === path) return sourceFile;
        }

        // Need to look for other files.
        const info = this.getOrCreateScriptInfoNotOpenedByClient(fileName, (project || this).currentDirectory, project ? project.directoryStructureHost : this.host);
        if (!info) return undefined;

        // Attach as source
        if (declarationInfo && ts.isString(declarationInfo.sourceMapFilePath) && info !== declarationInfo) {
            const sourceMapInfo = this.getScriptInfoForPath(declarationInfo.sourceMapFilePath);
            if (sourceMapInfo) {
                (sourceMapInfo.sourceInfos || (sourceMapInfo.sourceInfos = new ts.Set())).add(info.path);
            }
        }

        // Key doesnt matter since its only for text and lines
        if (info.cacheSourceFile) return info.cacheSourceFile.sourceFile;

        // Create sourceFileLike
        if (!info.sourceFileLike) {
            info.sourceFileLike = {
                get text() {
                    ts.Debug.fail("shouldnt need text");
                    return "";
                },
                getLineAndCharacterOfPosition: pos => {
                    const lineOffset = info.positionToLineOffset(pos);
                    return { line: lineOffset.line - 1, character: lineOffset.offset - 1 };
                },
                getPositionOfLineAndCharacter: (line, character, allowEdits) => info.lineOffsetToPosition(line + 1, character + 1, allowEdits)
            };
        }
        return info.sourceFileLike;
    }

    /*@internal*/
    setPerformanceEventHandler(performanceEventHandler: PerformanceEventHandler) {
        this.performanceEventHandler = performanceEventHandler;
    }

    setHostConfiguration(args: ts.server.protocol.ConfigureRequestArguments) {
        if (args.file) {
            const info = this.getScriptInfoForNormalizedPath(ts.server.toNormalizedPath(args.file));
            if (info) {
                info.setOptions(convertFormatOptions(args.formatOptions!), args.preferences);
                this.logger.info(`Host configuration update for file ${args.file}`);
            }
        }
        else {
            if (args.hostInfo !== undefined) {
                this.hostConfiguration.hostInfo = args.hostInfo;
                this.logger.info(`Host information ${args.hostInfo}`);
            }
            if (args.formatOptions) {
                this.hostConfiguration.formatCodeOptions = { ...this.hostConfiguration.formatCodeOptions, ...convertFormatOptions(args.formatOptions) };
                this.logger.info("Format host information updated");
            }
            if (args.preferences) {
                const {
                    lazyConfiguredProjectsFromExternalProject,
                    includePackageJsonAutoImports,
                } = this.hostConfiguration.preferences;

                this.hostConfiguration.preferences = { ...this.hostConfiguration.preferences, ...args.preferences };
                if (lazyConfiguredProjectsFromExternalProject && !this.hostConfiguration.preferences.lazyConfiguredProjectsFromExternalProject) {
                    // Load configured projects for external projects that are pending reload
                    this.configuredProjects.forEach(project => {
                        if (project.hasExternalProjectRef() &&
                            project.pendingReload === ts.ConfigFileProgramReloadLevel.Full &&
                            !this.pendingProjectUpdates.has(project.getProjectName())) {
                            project.updateGraph();
                        }
                    });
                }
                if (includePackageJsonAutoImports !== args.preferences.includePackageJsonAutoImports) {
                    this.invalidateProjectPackageJson(/*packageJsonPath*/ undefined);
                }
            }
            if (args.extraFileExtensions) {
                this.hostConfiguration.extraFileExtensions = args.extraFileExtensions;
                // We need to update the project structures again as it is possible that existing
                // project structure could have more or less files depending on extensions permitted
                this.reloadProjects();
                this.logger.info("Host file extension mappings updated");
            }

            if (args.watchOptions) {
                this.hostConfiguration.watchOptions = convertWatchOptions(args.watchOptions)?.watchOptions;
                this.logger.info(`Host watch options changed to ${JSON.stringify(this.hostConfiguration.watchOptions)}, it will be take effect for next watches.`);
            }
        }
    }

    /*@internal*/
    getWatchOptions(project: ts.server.Project) {
        return this.getWatchOptionsFromProjectWatchOptions(project.getWatchOptions());
    }

    /*@internal*/
    private getWatchOptionsFromProjectWatchOptions(projectOptions: ts.WatchOptions | undefined) {
        return projectOptions && this.hostConfiguration.watchOptions ?
            { ...this.hostConfiguration.watchOptions, ...projectOptions } :
            projectOptions || this.hostConfiguration.watchOptions;
    }

    closeLog() {
        this.logger.close();
    }

    /**
     * This function rebuilds the project for every file opened by the client
     * This does not reload contents of open files from disk. But we could do that if needed
     */
    reloadProjects() {
        this.logger.info("reload projects.");
        // If we want this to also reload open files from disk, we could do that,
        // but then we need to make sure we arent calling this function
        // (and would separate out below reloading of projects to be called when immediate reload is needed)
        // as there is no need to load contents of the files from the disk

        // Reload script infos
        this.filenameToScriptInfo.forEach(info => {
            if (this.openFiles.has(info.path)) return; // Skip open files
            if (!info.fileWatcher) return; // not watched file
            // Handle as if file is changed or deleted
            this.onSourceFileChanged(info, this.host.fileExists(info.fileName) ? ts.FileWatcherEventKind.Changed : ts.FileWatcherEventKind.Deleted);
        });
        // Cancel all project updates since we will be updating them now
        this.pendingProjectUpdates.forEach((_project, projectName) => {
            this.throttledOperations.cancel(projectName);
            this.pendingProjectUpdates.delete(projectName);
        });
        this.throttledOperations.cancel(ensureProjectForOpenFileSchedule);
        this.pendingEnsureProjectForOpenFiles = false;

        // Ensure everything is reloaded for cached configs
        this.configFileExistenceInfoCache.forEach(info => {
            if (info.config) info.config.reloadLevel = ts.ConfigFileProgramReloadLevel.Full;
        });

        // Reload Projects
        this.reloadConfiguredProjectForFiles(this.openFiles as ts.ESMap<ts.Path, ts.server.NormalizedPath | undefined>, /*clearSemanticCache*/ true, /*delayReload*/ false, ts.returnTrue, "User requested reload projects");
        this.externalProjects.forEach(project => {
            this.clearSemanticCache(project);
            project.updateGraph();
        });
        this.inferredProjects.forEach(project => this.clearSemanticCache(project));
        this.ensureProjectForOpenFiles();
    }

    /**
     * This function goes through all the openFiles and tries to file the config file for them.
     * If the config file is found and it refers to existing project, it reloads it either immediately
     * or schedules it for reload depending on delayReload option
     * If there is no existing project it just opens the configured project for the config file
     * reloadForInfo provides a way to filter out files to reload configured project for
     */
    private reloadConfiguredProjectForFiles<T>(openFiles: ts.ESMap<ts.Path, T> | undefined, clearSemanticCache: boolean, delayReload: boolean, shouldReloadProjectFor: (openFileValue: T) => boolean, reason: string) {
        const updatedProjects = new ts.Map<string, true>();
        const reloadChildProject = (child: ts.server.ConfiguredProject) => {
            if (!updatedProjects.has(child.canonicalConfigFilePath)) {
                updatedProjects.set(child.canonicalConfigFilePath, true);
                this.reloadConfiguredProject(child, reason, /*isInitialLoad*/ false, clearSemanticCache);
            }
        };
        // try to reload config file for all open files
        openFiles?.forEach((openFileValue, path) => {
            // Invalidate default config file name for open file
            this.configFileForOpenFiles.delete(path);
            // Filter out the files that need to be ignored
            if (!shouldReloadProjectFor(openFileValue)) {
                return;
            }

            const info = this.getScriptInfoForPath(path)!; // TODO: GH#18217
            ts.Debug.assert(info.isScriptOpen());
            // This tries to search for a tsconfig.json for the given file. If we found it,
            // we first detect if there is already a configured project created for it: if so,
            // we re- read the tsconfig file content and update the project only if we havent already done so
            // otherwise we create a new one.
            const configFileName = this.getConfigFileNameForFile(info);
            if (configFileName) {
                const project = this.findConfiguredProjectByProjectName(configFileName) || this.createConfiguredProject(configFileName);
                if (!updatedProjects.has(project.canonicalConfigFilePath)) {
                    updatedProjects.set(project.canonicalConfigFilePath, true);
                    if (delayReload) {
                        project.pendingReload = ts.ConfigFileProgramReloadLevel.Full;
                        project.pendingReloadReason = reason;
                        if (clearSemanticCache) this.clearSemanticCache(project);
                        this.delayUpdateProjectGraph(project);
                    }
                    else {
                        // reload from the disk
                        this.reloadConfiguredProject(project, reason, /*isInitialLoad*/ false, clearSemanticCache);
                        // If this project does not contain this file directly, reload the project till the reloaded project contains the script info directly
                        if (!projectContainsInfoDirectly(project, info)) {
                            const referencedProject = forEachResolvedProjectReferenceProject(
                                project,
                                info.path,
                                child => {
                                    reloadChildProject(child);
                                    return projectContainsInfoDirectly(child, info);
                                },
                                ProjectReferenceProjectLoadKind.FindCreate
                            );
                            if (referencedProject) {
                                // Reload the project's tree that is already present
                                forEachResolvedProjectReferenceProject(
                                    project,
                                    /*fileName*/ undefined,
                                    reloadChildProject,
                                    ProjectReferenceProjectLoadKind.Find
                                );
                            }
                        }
                    }
                }
            }
        });
    }

    /**
     * Remove the root of inferred project if script info is part of another project
     */
    private removeRootOfInferredProjectIfNowPartOfOtherProject(info: ts.server.ScriptInfo) {
        // If the script info is root of inferred project, it could only be first containing project
        // since info is added as root to the inferred project only when there are no other projects containing it
        // So when it is root of the inferred project and after project structure updates its now part
        // of multiple project it needs to be removed from that inferred project because:
        // - references in inferred project supersede the root part
        // - root / reference in non - inferred project beats root in inferred project

        // eg. say this is structure /a/b/a.ts /a/b/c.ts where c.ts references a.ts
        // When a.ts is opened, since there is no configured project/external project a.ts can be part of
        // a.ts is added as root to inferred project.
        // Now at time of opening c.ts, c.ts is also not aprt of any existing project,
        // so it will be added to inferred project as a root. (for sake of this example assume single inferred project is false)
        // So at this poing a.ts is part of first inferred project and second inferred project (of which c.ts is root)
        // And hence it needs to be removed from the first inferred project.
        ts.Debug.assert(info.containingProjects.length > 0);
        const firstProject = info.containingProjects[0];

        if (!firstProject.isOrphan() &&
            ts.server.isInferredProject(firstProject) &&
            firstProject.isRoot(info) &&
            ts.forEach(info.containingProjects, p => p !== firstProject && !p.isOrphan())) {
            firstProject.removeFile(info, /*fileExists*/ true, /*detachFromProject*/ true);
        }
    }

    /**
     * This function is to update the project structure for every inferred project.
     * It is called on the premise that all the configured projects are
     * up to date.
     * This will go through open files and assign them to inferred project if open file is not part of any other project
     * After that all the inferred project graphs are updated
     */
    private ensureProjectForOpenFiles() {
        this.logger.info("Before ensureProjectForOpenFiles:");
        this.printProjects();

        this.openFiles.forEach((projectRootPath, path) => {
            const info = this.getScriptInfoForPath(path as ts.Path)!;
            // collect all orphaned script infos from open files
            if (info.isOrphan()) {
                this.assignOrphanScriptInfoToInferredProject(info, projectRootPath);
            }
            else {
                // Or remove the root of inferred project if is referenced in more than one projects
                this.removeRootOfInferredProjectIfNowPartOfOtherProject(info);
            }
        });
        this.pendingEnsureProjectForOpenFiles = false;
        this.inferredProjects.forEach(updateProjectIfDirty);

        this.logger.info("After ensureProjectForOpenFiles:");
        this.printProjects();
    }

    /**
     * Open file whose contents is managed by the client
     * @param filename is absolute pathname
     * @param fileContent is a known version of the file content that is more up to date than the one on disk
     */
    openClientFile(fileName: string, fileContent?: string, scriptKind?: ts.ScriptKind, projectRootPath?: string): OpenConfiguredProjectResult {
        return this.openClientFileWithNormalizedPath(ts.server.toNormalizedPath(fileName), fileContent, scriptKind, /*hasMixedContent*/ false, projectRootPath ? ts.server.toNormalizedPath(projectRootPath) : undefined);
    }

    /*@internal*/
    getOriginalLocationEnsuringConfiguredProject(project: ts.server.Project, location: ts.DocumentPosition): ts.DocumentPosition | undefined {
        const isSourceOfProjectReferenceRedirect = project.isSourceOfProjectReferenceRedirect(location.fileName);
        const originalLocation = isSourceOfProjectReferenceRedirect ?
            location :
            project.getSourceMapper().tryGetSourcePosition(location);
        if (!originalLocation) return undefined;

        const { fileName } = originalLocation;
        const scriptInfo = this.getScriptInfo(fileName);
        if (!scriptInfo && !this.host.fileExists(fileName)) return undefined;

        const originalFileInfo: OriginalFileInfo = { fileName: ts.server.toNormalizedPath(fileName), path: this.toPath(fileName) };
        const configFileName = this.getConfigFileNameForFile(originalFileInfo);
        if (!configFileName) return undefined;

        let configuredProject: ts.server.ConfiguredProject | undefined = this.findConfiguredProjectByProjectName(configFileName);
        if (!configuredProject) {
            if (project.getCompilerOptions().disableReferencedProjectLoad) {
                // If location was a project reference redirect, then `location` and `originalLocation` are the same.
                if (isSourceOfProjectReferenceRedirect) {
                    return location;
                }

                // Otherwise, if we found `originalLocation` via a source map instead, then we check whether it's in
                // an open project.  If it is, we should search the containing project(s), even though the "default"
                // configured project isn't open.  However, if it's not in an open project, we need to stick with
                // `location` (i.e. the .d.ts file) because otherwise we'll miss the references in that file.
                return scriptInfo?.containingProjects.length
                    ? originalLocation
                    : location;
            }

            configuredProject = this.createAndLoadConfiguredProject(configFileName, `Creating project for original file: ${originalFileInfo.fileName}${location !== originalLocation ? " for location: " + location.fileName : ""}`);
        }
        updateProjectIfDirty(configuredProject);

        const projectContainsOriginalInfo = (project: ts.server.ConfiguredProject) => {
            const info = this.getScriptInfo(fileName);
            return info && projectContainsInfoDirectly(project, info);
        };

        if (configuredProject.isSolution() || !projectContainsOriginalInfo(configuredProject)) {
            // Find the project that is referenced from this solution that contains the script info directly
            configuredProject = forEachResolvedProjectReferenceProject(
                configuredProject,
                fileName,
                child => {
                    updateProjectIfDirty(child);
                    return projectContainsOriginalInfo(child) ? child : undefined;
                },
                ProjectReferenceProjectLoadKind.FindCreateLoad,
                `Creating project referenced in solution ${configuredProject.projectName} to find possible configured project for original file: ${originalFileInfo.fileName}${location !== originalLocation ? " for location: " + location.fileName : ""}`
            );
            if (!configuredProject) return undefined;
            if (configuredProject === project) return originalLocation;
        }

        // Keep this configured project as referenced from project
        addOriginalConfiguredProject(configuredProject);

        const originalScriptInfo = this.getScriptInfo(fileName);
        if (!originalScriptInfo || !originalScriptInfo.containingProjects.length) return undefined;

        // Add configured projects as referenced
        originalScriptInfo.containingProjects.forEach(project => {
            if (ts.server.isConfiguredProject(project)) {
                addOriginalConfiguredProject(project);
            }
        });
        return originalLocation;

        function addOriginalConfiguredProject(originalProject: ts.server.ConfiguredProject) {
            if (!project.originalConfiguredProjects) {
                project.originalConfiguredProjects = new ts.Set();
            }
            project.originalConfiguredProjects.add(originalProject.canonicalConfigFilePath);
        }
    }

    /** @internal */
    fileExists(fileName: ts.server.NormalizedPath): boolean {
        return !!this.getScriptInfoForNormalizedPath(fileName) || this.host.fileExists(fileName);
    }

    private findExternalProjectContainingOpenScriptInfo(info: ts.server.ScriptInfo): ts.server.ExternalProject | undefined {
        return ts.find(this.externalProjects, proj => {
            // Ensure project structure is up-to-date to check if info is present in external project
            updateProjectIfDirty(proj);
            return proj.containsScriptInfo(info);
        });
    }

    private getOrCreateOpenScriptInfo(fileName: ts.server.NormalizedPath, fileContent: string | undefined, scriptKind: ts.ScriptKind | undefined, hasMixedContent: boolean | undefined, projectRootPath: ts.server.NormalizedPath | undefined) {
        const info = this.getOrCreateScriptInfoOpenedByClientForNormalizedPath(fileName, projectRootPath ? this.getNormalizedAbsolutePath(projectRootPath) : this.currentDirectory, fileContent, scriptKind, hasMixedContent)!; // TODO: GH#18217
        this.openFiles.set(info.path, projectRootPath);
        return info;
    }

    private assignProjectToOpenedScriptInfo(info: ts.server.ScriptInfo): AssignProjectResult {
        let configFileName: ts.server.NormalizedPath | undefined;
        let configFileErrors: readonly ts.Diagnostic[] | undefined;
        let project: ts.server.ConfiguredProject | ts.server.ExternalProject | undefined = this.findExternalProjectContainingOpenScriptInfo(info);
        let retainProjects: ts.server.ConfiguredProject[] | ts.server.ConfiguredProject | undefined;
        let projectForConfigFileDiag: ts.server.ConfiguredProject | undefined;
        let defaultConfigProjectIsCreated = false;
        if (!project && this.serverMode === ts.LanguageServiceMode.Semantic) { // Checking semantic mode is an optimization
            configFileName = this.getConfigFileNameForFile(info);
            if (configFileName) {
                project = this.findConfiguredProjectByProjectName(configFileName);
                if (!project) {
                    project = this.createLoadAndUpdateConfiguredProject(configFileName, `Creating possible configured project for ${info.fileName} to open`);
                    defaultConfigProjectIsCreated = true;
                }
                else {
                    // Ensure project is ready to check if it contains opened script info
                    updateProjectIfDirty(project);
                }

                projectForConfigFileDiag = project.containsScriptInfo(info) ? project : undefined;
                retainProjects = project;

                // If this configured project doesnt contain script info but
                // it is solution with project references, try those project references
                if (!projectContainsInfoDirectly(project, info)) {
                    forEachResolvedProjectReferenceProject(
                        project,
                        info.path,
                        child => {
                            updateProjectIfDirty(child);
                            // Retain these projects
                            if (!ts.isArray(retainProjects)) {
                                retainProjects = [project as ts.server.ConfiguredProject, child];
                            }
                            else {
                                retainProjects.push(child);
                            }

                            // If script info belongs to this child project, use this as default config project
                            if (projectContainsInfoDirectly(child, info)) {
                                projectForConfigFileDiag = child;
                                return child;
                            }

                            // If this project uses the script info (even through project reference), if default project is not found, use this for configFileDiag
                            if (!projectForConfigFileDiag && child.containsScriptInfo(info)) {
                                projectForConfigFileDiag = child;
                            }
                        },
                        ProjectReferenceProjectLoadKind.FindCreateLoad,
                        `Creating project referenced in solution ${project.projectName} to find possible configured project for ${info.fileName} to open`
                    );
                }

                // Send the event only if the project got created as part of this open request and info is part of the project
                if (projectForConfigFileDiag) {
                    configFileName = projectForConfigFileDiag.getConfigFilePath();
                    if (projectForConfigFileDiag !== project || defaultConfigProjectIsCreated) {
                        configFileErrors = projectForConfigFileDiag.getAllProjectErrors();
                        this.sendConfigFileDiagEvent(projectForConfigFileDiag, info.fileName);
                    }
                }
                else {
                    // Since the file isnt part of configured project, do not send config file info
                    configFileName = undefined;
                }

                // Create ancestor configured project
                this.createAncestorProjects(info, project);
            }
        }

        // Project we have at this point is going to be updated since its either found through
        // - external project search, which updates the project before checking if info is present in it
        // - configured project - either created or updated to ensure we know correct status of info

        // At this point we need to ensure that containing projects of the info are uptodate
        // This will ensure that later question of info.isOrphan() will return correct answer
        // and we correctly create inferred project for the info
        info.containingProjects.forEach(updateProjectIfDirty);

        // At this point if file is part of any any configured or external project, then it would be present in the containing projects
        // So if it still doesnt have any containing projects, it needs to be part of inferred project
        if (info.isOrphan()) {
            // Even though this info did not belong to any of the configured projects, send the config file diag
            if (ts.isArray(retainProjects)) {
                retainProjects.forEach(project => this.sendConfigFileDiagEvent(project, info.fileName));
            }
            else if (retainProjects) {
                this.sendConfigFileDiagEvent(retainProjects, info.fileName);
            }
            ts.Debug.assert(this.openFiles.has(info.path));
            this.assignOrphanScriptInfoToInferredProject(info, this.openFiles.get(info.path));
        }
        ts.Debug.assert(!info.isOrphan());
        return { configFileName, configFileErrors, retainProjects };
    }

    private createAncestorProjects(info: ts.server.ScriptInfo, project: ts.server.ConfiguredProject) {
        // Skip if info is not part of default configured project
        if (!info.isAttached(project)) return;

        // Create configured project till project root
        while (true) {
            // Skip if project is not composite
            if (!project.isInitialLoadPending() &&
                (
                    !project.getCompilerOptions().composite ||
                    project.getCompilerOptions().disableSolutionSearching
                )
            ) return;

            // Get config file name
            const configFileName = this.getConfigFileNameForFile({
                fileName: project.getConfigFilePath(),
                path: info.path,
                configFileInfo: true
            });
            if (!configFileName) return;

            // find or delay load the project
            const ancestor = this.findConfiguredProjectByProjectName(configFileName) ||
                this.createConfiguredProjectWithDelayLoad(configFileName, `Creating project possibly referencing default composite project ${project.getProjectName()} of open file ${info.fileName}`);
            if (ancestor.isInitialLoadPending()) {
                // Set a potential project reference
                ancestor.setPotentialProjectReference(project.canonicalConfigFilePath);
            }
            project = ancestor;
        }
    }

    /*@internal*/
    loadAncestorProjectTree(forProjects?: ts.ReadonlyCollection<string>) {
        forProjects = forProjects || ts.mapDefinedEntries(
            this.configuredProjects,
            (key, project) => !project.isInitialLoadPending() ? [key, true] : undefined
        );

        const seenProjects = new ts.Set<ts.server.NormalizedPath>();
        // Work on array copy as we could add more projects as part of callback
        for (const project of ts.arrayFrom(this.configuredProjects.values())) {
            // If this project has potential project reference for any of the project we are loading ancestor tree for
            // load this project first
            if (forEachPotentialProjectReference(project, potentialRefPath => forProjects!.has(potentialRefPath))) {
                updateProjectIfDirty(project);
            }
            this.ensureProjectChildren(project, forProjects, seenProjects);
        }
    }

    private ensureProjectChildren(project: ts.server.ConfiguredProject, forProjects: ts.ReadonlyCollection<string>, seenProjects: ts.Set<ts.server.NormalizedPath>) {
        if (!ts.tryAddToSet(seenProjects, project.canonicalConfigFilePath)) return;

        // If this project disables child load ignore it
        if (project.getCompilerOptions().disableReferencedProjectLoad) return;

        const children = project.getCurrentProgram()?.getResolvedProjectReferences();
        if (!children) return;

        for (const child of children) {
            if (!child) continue;
            const referencedProject = ts.forEachResolvedProjectReference(child.references, ref => forProjects.has(ref.sourceFile.path) ? ref : undefined);
            if (!referencedProject) continue;

            // Load this project,
            const configFileName = ts.server.toNormalizedPath(child.sourceFile.fileName);
            const childProject = project.projectService.findConfiguredProjectByProjectName(configFileName) ||
                project.projectService.createAndLoadConfiguredProject(configFileName, `Creating project referenced by : ${project.projectName} as it references project ${referencedProject.sourceFile.fileName}`);
            updateProjectIfDirty(childProject);

            // Ensure children for this project
            this.ensureProjectChildren(childProject, forProjects, seenProjects);
        }
    }

    private cleanupAfterOpeningFile(toRetainConfigProjects: readonly ts.server.ConfiguredProject[] | ts.server.ConfiguredProject | undefined) {
        // This was postponed from closeOpenFile to after opening next file,
        // so that we can reuse the project if we need to right away
        this.removeOrphanConfiguredProjects(toRetainConfigProjects);

        // Remove orphan inferred projects now that we have reused projects
        // We need to create a duplicate because we cant guarantee order after removal
        for (const inferredProject of this.inferredProjects.slice()) {
            if (inferredProject.isOrphan()) {
                this.removeProject(inferredProject);
            }
        }

        // Delete the orphan files here because there might be orphan script infos (which are not part of project)
        // when some file/s were closed which resulted in project removal.
        // It was then postponed to cleanup these script infos so that they can be reused if
        // the file from that old project is reopened because of opening file from here.
        this.removeOrphanScriptInfos();
    }

    openClientFileWithNormalizedPath(fileName: ts.server.NormalizedPath, fileContent?: string, scriptKind?: ts.ScriptKind, hasMixedContent?: boolean, projectRootPath?: ts.server.NormalizedPath): OpenConfiguredProjectResult {
        const info = this.getOrCreateOpenScriptInfo(fileName, fileContent, scriptKind, hasMixedContent, projectRootPath);
        const { retainProjects, ...result } = this.assignProjectToOpenedScriptInfo(info);
        this.cleanupAfterOpeningFile(retainProjects);
        this.telemetryOnOpenFile(info);
        this.printProjects();
        return result;
    }

    private removeOrphanConfiguredProjects(toRetainConfiguredProjects: readonly ts.server.ConfiguredProject[] | ts.server.ConfiguredProject | undefined) {
        const toRemoveConfiguredProjects = new ts.Map(this.configuredProjects);
        const markOriginalProjectsAsUsed = (project: ts.server.Project) => {
            if (!project.isOrphan() && project.originalConfiguredProjects) {
                project.originalConfiguredProjects.forEach(
                    (_value, configuredProjectPath) => {
                        const project = this.getConfiguredProjectByCanonicalConfigFilePath(configuredProjectPath);
                        return project && retainConfiguredProject(project);
                    }
                );
            }
        };
        if (toRetainConfiguredProjects) {
            if (ts.isArray(toRetainConfiguredProjects)) {
                toRetainConfiguredProjects.forEach(retainConfiguredProject);
            }
            else {
                retainConfiguredProject(toRetainConfiguredProjects);
            }
        }

        // Do not remove configured projects that are used as original projects of other
        this.inferredProjects.forEach(markOriginalProjectsAsUsed);
        this.externalProjects.forEach(markOriginalProjectsAsUsed);
        this.configuredProjects.forEach(project => {
            // If project has open ref (there are more than zero references from external project/open file), keep it alive as well as any project it references
            if (project.hasOpenRef()) {
                retainConfiguredProject(project);
            }
            else if (toRemoveConfiguredProjects.has(project.canonicalConfigFilePath)) {
                // If the configured project for project reference has more than zero references, keep it alive
                forEachReferencedProject(
                    project,
                    ref => isRetained(ref) && retainConfiguredProject(project)
                );
            }
        });

        // Remove all the non marked projects
        toRemoveConfiguredProjects.forEach(project => this.removeProject(project));

        function isRetained(project: ts.server.ConfiguredProject) {
            return project.hasOpenRef() || !toRemoveConfiguredProjects.has(project.canonicalConfigFilePath);
        }

        function retainConfiguredProject(project: ts.server.ConfiguredProject) {
            if (toRemoveConfiguredProjects.delete(project.canonicalConfigFilePath)) {
                // Keep original projects used
                markOriginalProjectsAsUsed(project);
                // Keep all the references alive
                forEachReferencedProject(project, retainConfiguredProject);
            }
        }
    }

    private removeOrphanScriptInfos() {
        const toRemoveScriptInfos = new ts.Map(this.filenameToScriptInfo);
        this.filenameToScriptInfo.forEach(info => {
            // If script info is open or orphan, retain it and its dependencies
            if (!info.isScriptOpen() && info.isOrphan() && !info.isContainedByBackgroundProject()) {
                // Otherwise if there is any source info that is alive, this alive too
                if (!info.sourceMapFilePath) return;
                let sourceInfos: ts.Set<ts.Path> | undefined;
                if (ts.isString(info.sourceMapFilePath)) {
                    const sourceMapInfo = this.getScriptInfoForPath(info.sourceMapFilePath);
                    sourceInfos = sourceMapInfo && sourceMapInfo.sourceInfos;
                }
                else {
                    sourceInfos = info.sourceMapFilePath.sourceInfos;
                }
                if (!sourceInfos) return;
                if (!ts.forEachKey(sourceInfos, path => {
                    const info = this.getScriptInfoForPath(path);
                    return !!info && (info.isScriptOpen() || !info.isOrphan());
                })) {
                    return;
                }
            }

            // Retain this script info
            toRemoveScriptInfos.delete(info.path);
            if (info.sourceMapFilePath) {
                let sourceInfos: ts.Set<ts.Path> | undefined;
                if (ts.isString(info.sourceMapFilePath)) {
                    // And map file info and source infos
                    toRemoveScriptInfos.delete(info.sourceMapFilePath);
                    const sourceMapInfo = this.getScriptInfoForPath(info.sourceMapFilePath);
                    sourceInfos = sourceMapInfo && sourceMapInfo.sourceInfos;
                }
                else {
                    sourceInfos = info.sourceMapFilePath.sourceInfos;
                }
                if (sourceInfos) {
                    sourceInfos.forEach((_value, path) => toRemoveScriptInfos.delete(path));
                }
            }
        });

        toRemoveScriptInfos.forEach(info => {
            // if there are not projects that include this script info - delete it
            this.stopWatchingScriptInfo(info);
            this.deleteScriptInfo(info);
            info.closeSourceMapFileWatcher();
        });
    }

    private telemetryOnOpenFile(scriptInfo: ts.server.ScriptInfo): void {
        if (this.serverMode !== ts.LanguageServiceMode.Semantic || !this.eventHandler || !scriptInfo.isJavaScript() || !ts.addToSeen(this.allJsFilesForOpenFileTelemetry, scriptInfo.path)) {
            return;
        }

        const project = this.ensureDefaultProjectForFile(scriptInfo);
        if (!project.languageServiceEnabled) {
            return;
        }

        const sourceFile = project.getSourceFile(scriptInfo.path);
        const checkJs = !!sourceFile && !!sourceFile.checkJsDirective;
        this.eventHandler({ eventName: OpenFileInfoTelemetryEvent, data: { info: { checkJs } } });
    }

    /**
     * Close file whose contents is managed by the client
     * @param filename is absolute pathname
     */
    closeClientFile(uncheckedFileName: string): void;
    /*@internal*/
    closeClientFile(uncheckedFileName: string, skipAssignOrphanScriptInfosToInferredProject: true): boolean;
    closeClientFile(uncheckedFileName: string, skipAssignOrphanScriptInfosToInferredProject?: true) {
        const info = this.getScriptInfoForNormalizedPath(ts.server.toNormalizedPath(uncheckedFileName));
        const result = info ? this.closeOpenFile(info, skipAssignOrphanScriptInfosToInferredProject) : false;
        if (!skipAssignOrphanScriptInfosToInferredProject) {
            this.printProjects();
        }
        return result;
    }

    private collectChanges(
        lastKnownProjectVersions: ts.server.protocol.ProjectVersionInfo[],
        currentProjects: ts.server.Project[],
        includeProjectReferenceRedirectInfo: boolean | undefined,
        result: ts.server.ProjectFilesWithTSDiagnostics[]
        ): void {
        for (const proj of currentProjects) {
            const knownProject = ts.find(lastKnownProjectVersions, p => p.projectName === proj.getProjectName());
            result.push(proj.getChangesSinceVersion(knownProject && knownProject.version, includeProjectReferenceRedirectInfo));
        }
    }

    /* @internal */
    synchronizeProjectList(knownProjects: ts.server.protocol.ProjectVersionInfo[], includeProjectReferenceRedirectInfo?: boolean): ts.server.ProjectFilesWithTSDiagnostics[] {
        const files: ts.server.ProjectFilesWithTSDiagnostics[] = [];
        this.collectChanges(knownProjects, this.externalProjects, includeProjectReferenceRedirectInfo, files);
        this.collectChanges(knownProjects, ts.arrayFrom(this.configuredProjects.values()), includeProjectReferenceRedirectInfo, files);
        this.collectChanges(knownProjects, this.inferredProjects, includeProjectReferenceRedirectInfo, files);
        return files;
    }

    /* @internal */
    applyChangesInOpenFiles(openFiles: ts.Iterator<OpenFileArguments> | undefined, changedFiles?: ts.Iterator<ChangeFileArguments>, closedFiles?: string[]): void {
        let openScriptInfos: ts.server.ScriptInfo[] | undefined;
        let assignOrphanScriptInfosToInferredProject = false;
        if (openFiles) {
            while (true) {
                const iterResult = openFiles.next();
                if (iterResult.done) break;
                const file = iterResult.value;
                // Create script infos so we have the new content for all the open files before we do any updates to projects
                const info = this.getOrCreateOpenScriptInfo(
                    ts.server.toNormalizedPath(file.fileName),
                    file.content,
                    tryConvertScriptKindName(file.scriptKind!),
                    file.hasMixedContent,
                    file.projectRootPath ? ts.server.toNormalizedPath(file.projectRootPath) : undefined
                );
                (openScriptInfos || (openScriptInfos = [])).push(info);
            }
        }

        if (changedFiles) {
            while (true) {
                const iterResult = changedFiles.next();
                if (iterResult.done) break;
                const file = iterResult.value;
                const scriptInfo = this.getScriptInfo(file.fileName)!;
                ts.Debug.assert(!!scriptInfo);
                // Make edits to script infos and marks containing project as dirty
                this.applyChangesToFile(scriptInfo, file.changes);
            }
        }

        if (closedFiles) {
            for (const file of closedFiles) {
                // Close files, but dont assign projects to orphan open script infos, that part comes later
                assignOrphanScriptInfosToInferredProject = this.closeClientFile(file, /*skipAssignOrphanScriptInfosToInferredProject*/ true) || assignOrphanScriptInfosToInferredProject;
            }
        }

        // All the script infos now exist, so ok to go update projects for open files
        let retainProjects: readonly ts.server.ConfiguredProject[] | undefined;
        if (openScriptInfos) {
            retainProjects = ts.flatMap(openScriptInfos, info => this.assignProjectToOpenedScriptInfo(info).retainProjects);
        }

        // While closing files there could be open files that needed assigning new inferred projects, do it now
        if (assignOrphanScriptInfosToInferredProject) {
            this.assignOrphanScriptInfosToInferredProject();
        }

        if (openScriptInfos) {
            // Cleanup projects
            this.cleanupAfterOpeningFile(retainProjects);
            // Telemetry
            openScriptInfos.forEach(info => this.telemetryOnOpenFile(info));
            this.printProjects();
        }
        else if (ts.length(closedFiles)) {
            this.printProjects();
        }
    }

    /* @internal */
    applyChangesToFile(scriptInfo: ts.server.ScriptInfo, changes: ts.Iterator<ts.TextChange>) {
        while (true) {
            const iterResult = changes.next();
            if (iterResult.done) break;
            const change = iterResult.value;
            scriptInfo.editContent(change.span.start, change.span.start + change.span.length, change.newText);
        }
    }

    private closeConfiguredProjectReferencedFromExternalProject(configFile: ts.server.NormalizedPath) {
        const configuredProject = this.findConfiguredProjectByProjectName(configFile);
        if (configuredProject) {
            configuredProject.deleteExternalProjectReference();
            if (!configuredProject.hasOpenRef()) {
                this.removeProject(configuredProject);
                return;
            }
        }
    }

    closeExternalProject(uncheckedFileName: string): void {
        const fileName = ts.server.toNormalizedPath(uncheckedFileName);
        const configFiles = this.externalProjectToConfiguredProjectMap.get(fileName);
        if (configFiles) {
            for (const configFile of configFiles) {
                this.closeConfiguredProjectReferencedFromExternalProject(configFile);
            }
            this.externalProjectToConfiguredProjectMap.delete(fileName);
        }
        else {
            // close external project
            const externalProject = this.findExternalProjectByProjectName(uncheckedFileName);
            if (externalProject) {
                this.removeProject(externalProject);
            }
        }
    }

    openExternalProjects(projects: ts.server.protocol.ExternalProject[]): void {
        // record project list before the update
        const projectsToClose = ts.arrayToMap(this.externalProjects, p => p.getProjectName(), _ => true);
        ts.forEachKey(this.externalProjectToConfiguredProjectMap, externalProjectName => {
            projectsToClose.set(externalProjectName, true);
        });

        for (const externalProject of projects) {
            this.openExternalProject(externalProject);
            // delete project that is present in input list
            projectsToClose.delete(externalProject.projectFileName);
        }

        // close projects that were missing in the input list
        ts.forEachKey(projectsToClose, externalProjectName => {
            this.closeExternalProject(externalProjectName);
        });
    }

    /** Makes a filename safe to insert in a RegExp */
    private static readonly filenameEscapeRegexp = /[-\/\\^$*+?.()|[\]{}]/g;
    private static escapeFilenameForRegex(filename: string) {
        return filename.replace(this.filenameEscapeRegexp, "\\$&");
    }

    resetSafeList(): void {
        this.safelist = defaultTypeSafeList;
    }

    applySafeList(proj: ts.server.protocol.ExternalProject): ts.server.NormalizedPath[] {
        const { rootFiles } = proj;
        const typeAcquisition = proj.typeAcquisition!;
        ts.Debug.assert(!!typeAcquisition, "proj.typeAcquisition should be set by now");

        if (typeAcquisition.enable === false || typeAcquisition.disableFilenameBasedTypeAcquisition) {
            return [];
        }

        const typeAcqInclude = typeAcquisition.include || (typeAcquisition.include = []);
        const excludeRules: string[] = [];

        const normalizedNames = rootFiles.map(f => ts.normalizeSlashes(f.fileName)) as ts.server.NormalizedPath[];
        const excludedFiles: ts.server.NormalizedPath[] = [];

        for (const name of Object.keys(this.safelist)) {
            const rule = this.safelist[name];
            for (const root of normalizedNames) {
                if (rule.match.test(root)) {
                    this.logger.info(`Excluding files based on rule ${name} matching file '${root}'`);

                    // If the file matches, collect its types packages and exclude rules
                    if (rule.types) {
                        for (const type of rule.types) {
                            // Best-effort de-duping here - doesn't need to be unduplicated but
                            // we don't want the list to become a 400-element array of just 'kendo'
                            if (typeAcqInclude.indexOf(type) < 0) {
                                typeAcqInclude.push(type);
                            }
                        }
                    }

                    if (rule.exclude) {
                        for (const exclude of rule.exclude) {
                            const processedRule = root.replace(rule.match, (...groups: string[]) => {
                                return exclude.map(groupNumberOrString => {
                                    // RegExp group numbers are 1-based, but the first element in groups
                                    // is actually the original string, so it all works out in the end.
                                    if (typeof groupNumberOrString === "number") {
                                        if (!ts.isString(groups[groupNumberOrString])) {
                                            // Specification was wrong - exclude nothing!
                                            this.logger.info(`Incorrect RegExp specification in safelist rule ${name} - not enough groups`);
                                            // * can't appear in a filename; escape it because it's feeding into a RegExp
                                            return "\\*";
                                        }
                                        return ProjectService.escapeFilenameForRegex(groups[groupNumberOrString]);
                                    }
                                    return groupNumberOrString;
                                }).join("");
                            });

                            if (excludeRules.indexOf(processedRule) === -1) {
                                excludeRules.push(processedRule);
                            }
                        }
                    }
                    else {
                        // If not rules listed, add the default rule to exclude the matched file
                        const escaped = ProjectService.escapeFilenameForRegex(root);
                        if (excludeRules.indexOf(escaped) < 0) {
                            excludeRules.push(escaped);
                        }
                    }
                }
            }
        }

        const excludeRegexes = excludeRules.map(e => new RegExp(e, "i"));
        const filesToKeep: ts.server.protocol.ExternalFile[] = [];
        for (let i = 0; i < proj.rootFiles.length; i++) {
            if (excludeRegexes.some(re => re.test(normalizedNames[i]))) {
                excludedFiles.push(normalizedNames[i]);
            }
            else {
                let exclude = false;
                if (typeAcquisition.enable || typeAcquisition.enableAutoDiscovery) {
                    const baseName = ts.getBaseFileName(ts.toFileNameLowerCase(normalizedNames[i]));
                    if (ts.fileExtensionIs(baseName, "js")) {
                        const inferredTypingName = ts.removeFileExtension(baseName);
                        const cleanedTypingName = ts.removeMinAndVersionNumbers(inferredTypingName);
                        const typeName = this.legacySafelist.get(cleanedTypingName);
                        if (typeName !== undefined) {
                            this.logger.info(`Excluded '${normalizedNames[i]}' because it matched ${cleanedTypingName} from the legacy safelist`);
                            excludedFiles.push(normalizedNames[i]);
                            // *exclude* it from the project...
                            exclude = true;
                            // ... but *include* it in the list of types to acquire
                            // Same best-effort dedupe as above
                            if (typeAcqInclude.indexOf(typeName) < 0) {
                                typeAcqInclude.push(typeName);
                            }
                        }
                    }
                }
                if (!exclude) {
                    // Exclude any minified files that get this far
                    if (/^.+[\.-]min\.js$/.test(normalizedNames[i])) {
                        excludedFiles.push(normalizedNames[i]);
                    }
                    else {
                        filesToKeep.push(proj.rootFiles[i]);
                    }
                }
            }
        }
        proj.rootFiles = filesToKeep;
        return excludedFiles;
    }

    openExternalProject(proj: ts.server.protocol.ExternalProject): void {
        // typingOptions has been deprecated and is only supported for backward compatibility
        // purposes. It should be removed in future releases - use typeAcquisition instead.
        if (proj.typingOptions && !proj.typeAcquisition) {
            const typeAcquisition = ts.convertEnableAutoDiscoveryToEnable(proj.typingOptions);
            proj.typeAcquisition = typeAcquisition;
        }
        proj.typeAcquisition = proj.typeAcquisition || {};
        proj.typeAcquisition.include = proj.typeAcquisition.include || [];
        proj.typeAcquisition.exclude = proj.typeAcquisition.exclude || [];
        if (proj.typeAcquisition.enable === undefined) {
            proj.typeAcquisition.enable = ts.server.hasNoTypeScriptSource(proj.rootFiles.map(f => f.fileName));
        }

        const excludedFiles = this.applySafeList(proj);

        let tsConfigFiles: ts.server.NormalizedPath[] | undefined;
        const rootFiles: ts.server.protocol.ExternalFile[] = [];
        for (const file of proj.rootFiles) {
            const normalized = ts.server.toNormalizedPath(file.fileName);
            if (ts.server.getBaseConfigFileName(normalized)) {
                if (this.serverMode === ts.LanguageServiceMode.Semantic && this.host.fileExists(normalized)) {
                    (tsConfigFiles || (tsConfigFiles = [])).push(normalized);
                }
            }
            else {
                rootFiles.push(file);
            }
        }

        // sort config files to simplify comparison later
        if (tsConfigFiles) {
            tsConfigFiles.sort();
        }

        const externalProject = this.findExternalProjectByProjectName(proj.projectFileName);
        let exisingConfigFiles: string[] | undefined;
        if (externalProject) {
            externalProject.excludedFiles = excludedFiles;
            if (!tsConfigFiles) {
                const compilerOptions = convertCompilerOptions(proj.options);
                const watchOptionsAndErrors = convertWatchOptions(proj.options, externalProject.getCurrentDirectory());
                const lastFileExceededProgramSize = this.getFilenameForExceededTotalSizeLimitForNonTsFiles(proj.projectFileName, compilerOptions, proj.rootFiles, externalFilePropertyReader);
                if (lastFileExceededProgramSize) {
                    externalProject.disableLanguageService(lastFileExceededProgramSize);
                }
                else {
                    externalProject.enableLanguageService();
                }
                externalProject.setProjectErrors(watchOptionsAndErrors?.errors);
                // external project already exists and not config files were added - update the project and return;
                // The graph update here isnt postponed since any file open operation needs all updated external projects
                this.updateRootAndOptionsOfNonInferredProject(externalProject, proj.rootFiles, externalFilePropertyReader, compilerOptions, proj.typeAcquisition, proj.options.compileOnSave, watchOptionsAndErrors?.watchOptions);
                externalProject.updateGraph();
                return;
            }
            // some config files were added to external project (that previously were not there)
            // close existing project and later we'll open a set of configured projects for these files
            this.closeExternalProject(proj.projectFileName);
        }
        else if (this.externalProjectToConfiguredProjectMap.get(proj.projectFileName)) {
            // this project used to include config files
            if (!tsConfigFiles) {
                // config files were removed from the project - close existing external project which in turn will close configured projects
                this.closeExternalProject(proj.projectFileName);
            }
            else {
                // project previously had some config files - compare them with new set of files and close all configured projects that correspond to unused files
                const oldConfigFiles = this.externalProjectToConfiguredProjectMap.get(proj.projectFileName)!;
                let iNew = 0;
                let iOld = 0;
                while (iNew < tsConfigFiles.length && iOld < oldConfigFiles.length) {
                    const newConfig = tsConfigFiles[iNew];
                    const oldConfig = oldConfigFiles[iOld];
                    if (oldConfig < newConfig) {
                        this.closeConfiguredProjectReferencedFromExternalProject(oldConfig);
                        iOld++;
                    }
                    else if (oldConfig > newConfig) {
                        iNew++;
                    }
                    else {
                        // record existing config files so avoid extra add-refs
                        (exisingConfigFiles || (exisingConfigFiles = [])).push(oldConfig);
                        iOld++;
                        iNew++;
                    }
                }
                for (let i = iOld; i < oldConfigFiles.length; i++) {
                    // projects for all remaining old config files should be closed
                    this.closeConfiguredProjectReferencedFromExternalProject(oldConfigFiles[i]);
                }
            }
        }
        if (tsConfigFiles) {
            // store the list of tsconfig files that belong to the external project
            this.externalProjectToConfiguredProjectMap.set(proj.projectFileName, tsConfigFiles);
            for (const tsconfigFile of tsConfigFiles) {
                let project = this.findConfiguredProjectByProjectName(tsconfigFile);
                if (!project) {
                    // errors are stored in the project, do not need to update the graph
                    project = this.getHostPreferences().lazyConfiguredProjectsFromExternalProject ?
                        this.createConfiguredProjectWithDelayLoad(tsconfigFile, `Creating configured project in external project: ${proj.projectFileName}`) :
                        this.createLoadAndUpdateConfiguredProject(tsconfigFile, `Creating configured project in external project: ${proj.projectFileName}`);
                }
                if (project && !ts.contains(exisingConfigFiles, tsconfigFile)) {
                    // keep project alive even if no documents are opened - its lifetime is bound to the lifetime of containing external project
                    project.addExternalProjectReference();
                }
            }
        }
        else {
            // no config files - remove the item from the collection
            // Create external project and update its graph, do not delay update since
            // any file open operation needs all updated external projects
            this.externalProjectToConfiguredProjectMap.delete(proj.projectFileName);
            const project = this.createExternalProject(proj.projectFileName, rootFiles, proj.options, proj.typeAcquisition, excludedFiles);
            project.updateGraph();
        }
    }

    hasDeferredExtension() {
        for (const extension of this.hostConfiguration.extraFileExtensions!) { // TODO: GH#18217
            if (extension.scriptKind === ts.ScriptKind.Deferred) {
                return true;
            }
        }

        return false;
    }

    /*@internal*/
    requestEnablePlugin(project: ts.server.Project, pluginConfigEntry: ts.PluginImport, searchPaths: string[], pluginConfigOverrides: ts.Map<any> | undefined) {
        if (!this.host.importPlugin && !this.host.require) {
            this.logger.info("Plugins were requested but not running in environment that supports 'require'. Nothing will be loaded");
            return;
        }

        this.logger.info(`Enabling plugin ${pluginConfigEntry.name} from candidate paths: ${searchPaths.join(",")}`);
        if (!pluginConfigEntry.name || ts.parsePackageName(pluginConfigEntry.name).rest) {
            this.logger.info(`Skipped loading plugin ${pluginConfigEntry.name || JSON.stringify(pluginConfigEntry)} because only package name is allowed plugin name`);
            return;
        }

        // If the host supports dynamic import, begin enabling the plugin asynchronously.
        if (this.host.importPlugin) {
            const importPromise = project.beginEnablePluginAsync(pluginConfigEntry, searchPaths, pluginConfigOverrides);
            this.pendingPluginEnablements ??= new ts.Map();
            let promises = this.pendingPluginEnablements.get(project);
            if (!promises) this.pendingPluginEnablements.set(project, promises = []);
            promises.push(importPromise);
            return;
        }

        // Otherwise, load the plugin using `require`
        project.endEnablePlugin(project.beginEnablePluginSync(pluginConfigEntry, searchPaths, pluginConfigOverrides));
    }

    /* @internal */
    hasNewPluginEnablementRequests() {
        return !!this.pendingPluginEnablements;
    }

    /* @internal */
    hasPendingPluginEnablements() {
        return !!this.currentPluginEnablementPromise;
    }

    /**
     * Waits for any ongoing plugin enablement requests to complete.
     */
    /* @internal */
    async waitForPendingPlugins() {
        while (this.currentPluginEnablementPromise) {
            await this.currentPluginEnablementPromise;
        }
    }

    /**
     * Starts enabling any requested plugins without waiting for the result.
     */
    /* @internal */
    enableRequestedPlugins() {
        if (this.pendingPluginEnablements) {
            void this.enableRequestedPluginsAsync();
        }
    }

    private async enableRequestedPluginsAsync() {
        if (this.currentPluginEnablementPromise) {
            // If we're already enabling plugins, wait for any existing operations to complete
            await this.waitForPendingPlugins();
        }

        // Skip if there are no new plugin enablement requests
        if (!this.pendingPluginEnablements) {
            return;
        }

        // Consume the pending plugin enablement requests
        const entries = ts.arrayFrom(this.pendingPluginEnablements.entries());
        this.pendingPluginEnablements = undefined;

        // Start processing the requests, keeping track of the promise for the operation so that
        // project consumers can potentially wait for the plugins to load.
        this.currentPluginEnablementPromise = this.enableRequestedPluginsWorker(entries);
        await this.currentPluginEnablementPromise;
    }

    private async enableRequestedPluginsWorker(pendingPlugins: [ts.server.Project, Promise<ts.server.BeginEnablePluginResult>[]][]) {
        // This should only be called from `enableRequestedPluginsAsync`, which ensures this precondition is met.
        ts.Debug.assert(this.currentPluginEnablementPromise === undefined);

        // Process all pending plugins, partitioned by project. This way a project with few plugins doesn't need to wait
        // on a project with many plugins.
        await Promise.all(ts.map(pendingPlugins, ([project, promises]) => this.enableRequestedPluginsForProjectAsync(project, promises)));

        // Clear the pending operation and notify the client that projects have been updated.
        this.currentPluginEnablementPromise = undefined;
        this.sendProjectsUpdatedInBackgroundEvent();
    }

    private async enableRequestedPluginsForProjectAsync(project: ts.server.Project, promises: Promise<ts.server.BeginEnablePluginResult>[]) {
        // Await all pending plugin imports. This ensures all requested plugin modules are fully loaded
        // prior to patching the language service, and that any promise rejections are observed.
        const results = await Promise.all(promises);
        if (project.isClosed()) {
            // project is not alive, so don't enable plugins.
            return;
        }

        for (const result of results) {
            project.endEnablePlugin(result);
        }

        // Plugins may have modified external files, so mark the project as dirty.
        this.delayUpdateProjectGraph(project);
    }

    configurePlugin(args: ts.server.protocol.ConfigurePluginRequestArguments) {
        // For any projects that already have the plugin loaded, configure the plugin
        this.forEachEnabledProject(project => project.onPluginConfigurationChanged(args.pluginName, args.configuration));

        // Also save the current configuration to pass on to any projects that are yet to be loaded.
        // If a plugin is configured twice, only the latest configuration will be remembered.
        this.currentPluginConfigOverrides = this.currentPluginConfigOverrides || new ts.Map();
        this.currentPluginConfigOverrides.set(args.pluginName, args.configuration);
    }

    /*@internal*/
    getPackageJsonsVisibleToFile(fileName: string, rootDir?: string): readonly ts.ProjectPackageJsonInfo[] {
        const packageJsonCache = this.packageJsonCache;
        const rootPath = rootDir && this.toPath(rootDir);
        const filePath = this.toPath(fileName);
        const result: ts.ProjectPackageJsonInfo[] = [];
        const processDirectory = (directory: ts.Path): boolean | undefined => {
            switch (packageJsonCache.directoryHasPackageJson(directory)) {
                // Sync and check same directory again
                case ts.Ternary.Maybe:
                    packageJsonCache.searchDirectoryAndAncestors(directory);
                    return processDirectory(directory);
                // Check package.json
                case ts.Ternary.True:
                    const packageJsonFileName = ts.combinePaths(directory, "package.json");
                    this.watchPackageJsonFile(packageJsonFileName as ts.Path);
                    const info = packageJsonCache.getInDirectory(directory);
                    if (info) result.push(info);
            }
            if (rootPath && rootPath === directory) {
                return true;
            }
        };

        ts.forEachAncestorDirectory(ts.getDirectoryPath(filePath), processDirectory);
        return result;
    }

    /*@internal*/
    getNearestAncestorDirectoryWithPackageJson(fileName: string): string | undefined {
        return ts.forEachAncestorDirectory(fileName, directory => {
            switch (this.packageJsonCache.directoryHasPackageJson(this.toPath(directory))) {
                case ts.Ternary.True: return directory;
                case ts.Ternary.False: return undefined;
                case ts.Ternary.Maybe:
                    return this.host.fileExists(ts.combinePaths(directory, "package.json"))
                        ? directory
                        : undefined;
            }
        });
    }

    /*@internal*/
    private watchPackageJsonFile(path: ts.Path) {
        const watchers = this.packageJsonFilesMap || (this.packageJsonFilesMap = new ts.Map());
        if (!watchers.has(path)) {
            this.invalidateProjectPackageJson(path);
            watchers.set(path, this.watchFactory.watchFile(
                path,
                (fileName, eventKind) => {
                    const path = this.toPath(fileName);
                    switch (eventKind) {
                        case ts.FileWatcherEventKind.Created:
                            return ts.Debug.fail();
                        case ts.FileWatcherEventKind.Changed:
                            this.packageJsonCache.addOrUpdate(path);
                            this.invalidateProjectPackageJson(path);
                            break;
                        case ts.FileWatcherEventKind.Deleted:
                            this.packageJsonCache.delete(path);
                            this.invalidateProjectPackageJson(path);
                            watchers.get(path)!.close();
                            watchers.delete(path);
                    }
                },
                ts.PollingInterval.Low,
                this.hostConfiguration.watchOptions,
                ts.WatchType.PackageJson,
            ));
        }
    }

    /*@internal*/
    private onAddPackageJson(path: ts.Path) {
        this.packageJsonCache.addOrUpdate(path);
        this.watchPackageJsonFile(path);
    }

    /*@internal*/
    includePackageJsonAutoImports(): ts.PackageJsonAutoImportPreference {
        switch (this.hostConfiguration.preferences.includePackageJsonAutoImports) {
            case "on": return ts.PackageJsonAutoImportPreference.On;
            case "off": return ts.PackageJsonAutoImportPreference.Off;
            default: return ts.PackageJsonAutoImportPreference.Auto;
        }
    }

    /*@internal*/
    private invalidateProjectPackageJson(packageJsonPath: ts.Path | undefined) {
        this.configuredProjects.forEach(invalidate);
        this.inferredProjects.forEach(invalidate);
        this.externalProjects.forEach(invalidate);
        function invalidate(project: ts.server.Project) {
            if (packageJsonPath) {
                project.onPackageJsonChange(packageJsonPath);
            }
            else {
                project.onAutoImportProviderSettingsChanged();
            }
        }
    }

    /*@internal*/
    getIncompleteCompletionsCache() {
        return this.incompleteCompletionsCache ||= createIncompleteCompletionsCache();
    }
}

function createIncompleteCompletionsCache(): ts.IncompleteCompletionsCache {
    let info: ts.CompletionInfo | undefined;
    return {
        get() {
            return info;
        },
        set(newInfo) {
            info = newInfo;
        },
        clear() {
            info = undefined;
        }
    };
}

/* @internal */
export type ScriptInfoOrConfig = ts.server.ScriptInfo | ts.TsConfigSourceFile;
/* @internal */
export function isConfigFile(config: ScriptInfoOrConfig): config is ts.TsConfigSourceFile {
    return (config as ts.TsConfigSourceFile).kind !== undefined;
}

function printProjectWithoutFileNames(project: ts.server.Project) {
    project.print(/*writeProjectFileNames*/ false);
}
