Info 0    [00:00:09.000] Provided types map file "/a/lib/typesMap.json" doesn't exist
Before request
//// [/jsconfig.json]
{"compilerOptions":{"allowJs":true,"allowSyntheticDefaultImports":true,"maxNodeModuleJsDepth":2,"skipLibCheck":true,"noEmit":true}}

//// [/b.ts]
12

//// [/a.js]
1


Info 1    [00:00:10.000] request:
    {
      "command": "open",
      "arguments": {
        "file": "/a.js"
      },
      "seq": 1,
      "type": "request"
    }
Info 2    [00:00:11.000] Search path: /
Info 3    [00:00:12.000] For info: /a.js :: Config file name: /jsconfig.json
Info 4    [00:00:13.000] Creating configuration project /jsconfig.json
Info 5    [00:00:14.000] FileWatcher:: Added:: WatchInfo: /jsconfig.json 2000 undefined Project: /jsconfig.json WatchType: Config file
Info 6    [00:00:15.000] event:
    {"seq":0,"type":"event","event":"projectLoadingStart","body":{"projectName":"/jsconfig.json","reason":"Creating possible configured project for /a.js to open"}}
Info 7    [00:00:16.000] Config: /jsconfig.json : {
 "rootNames": [
  "/a.js",
  "/b.ts"
 ],
 "options": {
  "allowJs": true,
  "maxNodeModuleJsDepth": 2,
  "allowSyntheticDefaultImports": true,
  "skipLibCheck": true,
  "noEmit": true,
  "configFilePath": "/jsconfig.json"
 }
}
Info 8    [00:00:17.000] DirectoryWatcher:: Added:: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 9    [00:00:18.000] Elapsed:: *ms DirectoryWatcher:: Added:: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 10   [00:00:19.000] FileWatcher:: Added:: WatchInfo: /b.ts 500 undefined WatchType: Closed Script info
Info 11   [00:00:20.000] Starting updateGraphWorker: Project: /jsconfig.json
Info 12   [00:00:21.000] FileWatcher:: Added:: WatchInfo: /a/lib/lib.d.ts 500 undefined Project: /jsconfig.json WatchType: Missing file
Info 13   [00:00:22.000] Finishing updateGraphWorker: Project: /jsconfig.json Version: 1 structureChanged: true structureIsReused:: Not Elapsed:: *ms
Info 14   [00:00:23.000] Project '/jsconfig.json' (Configured)
Info 15   [00:00:24.000] 	Files (2)
	/a.js SVC-1-0 "1"
	/b.ts Text-1 "12"


	a.js
	  Matched by default include pattern '**/*'
	b.ts
	  Matched by default include pattern '**/*'

Info 16   [00:00:25.000] -----------------------------------------------
TI:: Creating typing installer

PolledWatches::
/a/lib/lib.d.ts: *new*
  {"pollingInterval":500}

FsWatches::
/jsconfig.json: *new*
  {}
/b.ts: *new*
  {}

FsWatchesRecursive::
/: *new*
  {}

TI:: [00:00:26.000] Global cache location '/a/data/', safe file path '/safeList.json', types map path /typesMap.json
TI:: [00:00:27.000] Processing cache location '/a/data/'
TI:: [00:00:28.000] Trying to find '/a/data/package.json'...
TI:: [00:00:29.000] Finished processing cache location '/a/data/'
TI:: [00:00:30.000] Npm config file: /a/data/package.json
TI:: [00:00:31.000] Npm config file: '/a/data/package.json' is missing, creating new one...
Info 17   [00:00:34.000] DirectoryWatcher:: Triggered with a :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 18   [00:00:35.000] Scheduled: /jsconfig.json
Info 19   [00:00:36.000] Scheduled: *ensureProjectForOpenFiles*
Info 20   [00:00:37.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 21   [00:00:40.000] DirectoryWatcher:: Triggered with a/data :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 22   [00:00:41.000] Scheduled: /jsconfig.json, Cancelled earlier one
Info 23   [00:00:42.000] Scheduled: *ensureProjectForOpenFiles*, Cancelled earlier one
Info 24   [00:00:43.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a/data :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 25   [00:00:46.000] DirectoryWatcher:: Triggered with a/data/package.json :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 26   [00:00:47.000] Config: /jsconfig.json Detected new package.json: a/data/package.json
Info 27   [00:00:48.000] FileWatcher:: Added:: WatchInfo: /a/data/package.json 250 undefined WatchType: package.json file
Info 28   [00:00:49.000] Project: /jsconfig.json Detected file add/remove of non supported extension: a/data/package.json
Info 29   [00:00:50.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a/data/package.json :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
TI:: [00:00:51.000] Updating types-registry npm package...
TI:: [00:00:52.000] npm install --ignore-scripts types-registry@latest
Info 30   [00:00:57.000] DirectoryWatcher:: Triggered with a/data/node_modules :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 31   [00:00:58.000] Scheduled: /jsconfig.json, Cancelled earlier one
Info 32   [00:00:59.000] Scheduled: *ensureProjectForOpenFiles*, Cancelled earlier one
Info 33   [00:01:00.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a/data/node_modules :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 34   [00:01:02.000] DirectoryWatcher:: Triggered with a/data/node_modules/types-registry :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 35   [00:01:03.000] Scheduled: /jsconfig.json, Cancelled earlier one
Info 36   [00:01:04.000] Scheduled: *ensureProjectForOpenFiles*, Cancelled earlier one
Info 37   [00:01:05.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a/data/node_modules/types-registry :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 38   [00:01:07.000] DirectoryWatcher:: Triggered with a/data/node_modules/types-registry/index.json :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
Info 39   [00:01:08.000] Project: /jsconfig.json Detected file add/remove of non supported extension: a/data/node_modules/types-registry/index.json
Info 40   [00:01:09.000] Elapsed:: *ms DirectoryWatcher:: Triggered with a/data/node_modules/types-registry/index.json :: WatchInfo:  1 undefined Config: /jsconfig.json WatchType: Wild card directory
TI:: [00:01:10.000] TI:: Updated types-registry npm package
TI:: typing installer creation complete
//// [/a/data/package.json]
{ "private": true }

//// [/a/data/node_modules/types-registry/index.json]
{
 "entries": {}
}


PolledWatches::
/a/lib/lib.d.ts:
  {"pollingInterval":500}

FsWatches::
/jsconfig.json:
  {}
/b.ts:
  {}
/a/data/package.json: *new*
  {}

FsWatchesRecursive::
/:
  {}

TI:: [00:01:11.000] Got install request {"projectName":"/jsconfig.json","fileNames":["/a.js","/b.ts"],"compilerOptions":{"allowJs":true,"maxNodeModuleJsDepth":2,"allowSyntheticDefaultImports":true,"skipLibCheck":true,"noEmit":true,"configFilePath":"/jsconfig.json","allowNonTsExtensions":true},"typeAcquisition":{"enable":true,"include":[],"exclude":[]},"unresolvedImports":[],"projectRootPath":"/","cachePath":"/a/data/","kind":"discover"}
TI:: [00:01:12.000] Request specifies cache path '/a/data/', loading cached information...
TI:: [00:01:13.000] Processing cache location '/a/data/'
TI:: [00:01:14.000] Cache location was already processed...
TI:: [00:01:15.000] Failed to load safelist from types map file '/typesMap.json'
TI:: [00:01:16.000] Explicitly included types: []
TI:: [00:01:17.000] Inferred typings from unresolved imports: []
TI:: [00:01:18.000] Result: {"cachedTypingPaths":[],"newTypingNames":[],"filesToWatch":["/bower_components","/node_modules"]}
TI:: [00:01:19.000] Finished typings discovery: {"cachedTypingPaths":[],"newTypingNames":[],"filesToWatch":["/bower_components","/node_modules"]}
TI:: [00:01:20.000] DirectoryWatcher:: Added:: WatchInfo: /bower_components
TI:: [00:01:21.000] DirectoryWatcher:: Added:: WatchInfo: /bower_components 1 undefined Project: /jsconfig.json watcher already invoked: false
TI:: [00:01:22.000] Elapsed:: *ms DirectoryWatcher:: Added:: WatchInfo: /bower_components 1 undefined Project: /jsconfig.json watcher already invoked: false
TI:: [00:01:23.000] DirectoryWatcher:: Added:: WatchInfo: /node_modules
TI:: [00:01:24.000] DirectoryWatcher:: Added:: WatchInfo: /node_modules 1 undefined Project: /jsconfig.json watcher already invoked: false
TI:: [00:01:25.000] Elapsed:: *ms DirectoryWatcher:: Added:: WatchInfo: /node_modules 1 undefined Project: /jsconfig.json watcher already invoked: false
TI:: [00:01:26.000] Sending response:
    {"projectName":"/jsconfig.json","typeAcquisition":{"enable":true,"include":[],"exclude":[]},"compilerOptions":{"allowJs":true,"maxNodeModuleJsDepth":2,"allowSyntheticDefaultImports":true,"skipLibCheck":true,"noEmit":true,"configFilePath":"/jsconfig.json","allowNonTsExtensions":true},"typings":[],"unresolvedImports":[],"kind":"action::set"}
TI:: [00:01:27.000] No new typings were requested as a result of typings discovery
Info 41   [00:01:28.000] event:
    {"seq":0,"type":"event","event":"projectLoadingFinish","body":{"projectName":"/jsconfig.json"}}
Info 42   [00:01:29.000] event:
    {"seq":0,"type":"event","event":"telemetry","body":{"telemetryEventName":"projectInfo","payload":{"projectId":"d3f7418c3d4888d0a51e42716b5a330dab4da64c452eebe918c1e0e634d8ede1","fileStats":{"js":1,"jsSize":1,"jsx":0,"jsxSize":0,"ts":1,"tsSize":2,"tsx":0,"tsxSize":0,"dts":0,"dtsSize":0,"deferred":0,"deferredSize":0},"compilerOptions":{"allowJs":true,"maxNodeModuleJsDepth":2,"allowSyntheticDefaultImports":true,"skipLibCheck":true,"noEmit":true},"typeAcquisition":{"enable":true,"include":false,"exclude":false},"extends":false,"files":false,"include":false,"exclude":false,"compileOnSave":false,"configFileName":"jsconfig.json","projectType":"configured","languageServiceEnabled":true,"version":"FakeVersion"}}}
Info 43   [00:01:30.000] Starting updateGraphWorker: Project: /jsconfig.json
Info 44   [00:01:31.000] Finishing updateGraphWorker: Project: /jsconfig.json Version: 2 structureChanged: false structureIsReused:: Not Elapsed:: *ms
Info 45   [00:01:32.000] Same program as before
Info 46   [00:01:33.000] event:
    {"seq":0,"type":"event","event":"configFileDiag","body":{"triggerFile":"/a.js","configFile":"/jsconfig.json","diagnostics":[{"text":"File '/a/lib/lib.d.ts' not found.\n  The file is in the program because:\n    Default library for target 'es5'","code":6053,"category":"error"},{"text":"Cannot find global type 'Array'.","code":2318,"category":"error"},{"text":"Cannot find global type 'Boolean'.","code":2318,"category":"error"},{"text":"Cannot find global type 'Function'.","code":2318,"category":"error"},{"text":"Cannot find global type 'IArguments'.","code":2318,"category":"error"},{"text":"Cannot find global type 'Number'.","code":2318,"category":"error"},{"text":"Cannot find global type 'Object'.","code":2318,"category":"error"},{"text":"Cannot find global type 'RegExp'.","code":2318,"category":"error"},{"text":"Cannot find global type 'String'.","code":2318,"category":"error"}]}}
Info 47   [00:01:34.000] Project '/jsconfig.json' (Configured)
Info 47   [00:01:35.000] 	Files (2)

Info 47   [00:01:36.000] -----------------------------------------------
Info 47   [00:01:37.000] Open files: 
Info 47   [00:01:38.000] 	FileName: /a.js ProjectRootPath: undefined
Info 47   [00:01:39.000] 		Projects: /jsconfig.json
Info 47   [00:01:40.000] response:
    {
      "responseRequired": false
    }
After request

PolledWatches::
/a/lib/lib.d.ts:
  {"pollingInterval":500}
/bower_components: *new*
  {"pollingInterval":500}
/node_modules: *new*
  {"pollingInterval":500}

FsWatches::
/jsconfig.json:
  {}
/b.ts:
  {}
/a/data/package.json:
  {}

FsWatchesRecursive::
/:
  {}
