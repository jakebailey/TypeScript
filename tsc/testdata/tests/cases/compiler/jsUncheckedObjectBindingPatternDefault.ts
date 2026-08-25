// @allowJs: true
// @checkJs: false
// @noEmit: true
// @strict: true

// @filename: /unchecked.js
export const unchecked = ({
    required,
    optional = false,
} = {}) => {};

// @filename: /checked.js
// @ts-check
export const checked = ({
    required,
    optional = false,
} = {}) => {};

// @filename: /main.ts
import { checked } from "./checked";
import { unchecked } from "./unchecked";

unchecked({ required: "value" });
checked({ required: "value" });
