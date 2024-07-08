//// [tests/cases/conformance/jsx/jsxReactReferencedImport.ts] ////

//// [file.tsx]
/// <reference path="/.lib/react16.d.ts" />

import React from 'react';

export function Component1() {
  return <div />;
}


//// [file.js]
/// <reference path="react16.d.ts" />
import React from 'react';
export function Component1() {
    return React.createElement("div", null);
}
