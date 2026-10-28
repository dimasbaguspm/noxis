import * as monaco from "monaco-editor/editor/editor.api.js";
import editorWorker from "monaco-editor/editor/editor.worker.js?worker";

// Editor features, curated: editor.api alone ships no UI features (no suggest widget), while editor.main drags in all ~90 language definitions.
import "monaco-editor/editor/browser/coreCommands.js";
import "monaco-editor/editor/contrib/bracketMatching/browser/bracketMatching.js";
import "monaco-editor/editor/contrib/clipboard/browser/clipboard.js";
import "monaco-editor/editor/contrib/comment/browser/comment.js";
import "monaco-editor/editor/contrib/contextmenu/browser/contextmenu.js";
import "monaco-editor/editor/contrib/cursorUndo/browser/cursorUndo.js";
import "monaco-editor/editor/contrib/find/browser/findController.js";
import "monaco-editor/editor/contrib/folding/browser/folding.js";
import "monaco-editor/editor/contrib/hover/browser/hoverContribution.js";
import "monaco-editor/editor/contrib/indentation/browser/indentation.js";
import "monaco-editor/editor/contrib/linesOperations/browser/linesOperations.js";
import "monaco-editor/editor/contrib/multicursor/browser/multicursor.js";
import "monaco-editor/editor/contrib/parameterHints/browser/parameterHints.js";
import "monaco-editor/editor/contrib/snippet/browser/snippetController2.js";
import "monaco-editor/editor/contrib/suggest/browser/suggestController.js";
import "monaco-editor/editor/contrib/wordHighlighter/browser/wordHighlighter.js";
import "monaco-editor/editor/contrib/wordOperations/browser/wordOperations.js";
import "monaco-editor/editor/contrib/wordPartOperations/browser/wordPartOperations.js";

// The editor uses SQL dialects and a small MongoDB JSON grammar without loading the validation worker.
import "monaco-editor/languages/definitions/sql/register.js";
import "monaco-editor/languages/definitions/pgsql/register.js";
import "monaco-editor/languages/definitions/mysql/register.js";

const MONGO_LANGUAGE = "mongo";

monaco.languages.register({
  id: MONGO_LANGUAGE,
  extensions: [".json"],
  aliases: ["MongoDB", "json"],
});
monaco.languages.setMonarchTokensProvider(MONGO_LANGUAGE, {
  operators: [
    "$eq",
    "$ne",
    "$gt",
    "$gte",
    "$lt",
    "$lte",
    "$in",
    "$nin",
    "$exists",
    "$regex",
    "$and",
    "$or",
    "$not",
    "$set",
    "$unset",
    "$inc",
    "$push",
    "$pull",
    "$addToSet",
    "$match",
    "$group",
    "$project",
  ],
  tokenizer: {
    root: [
      [/"\$?[a-zA-Z0-9_.]*"/, "string"],
      [/\$[a-zA-Z][a-zA-Z0-9]*/, "keyword"],
      [/-?\d+(\.\d+)?([eE][-+]?\d+)?/, "number"],
      [/\b(true|false|null)\b/, "keyword"],
      [/[{}[\],:]/, "delimiter"],
    ],
  },
});
monaco.languages.setLanguageConfiguration(MONGO_LANGUAGE, {
  brackets: [
    ["{", "}"],
    ["[", "]"],
  ],
  autoClosingPairs: [
    { open: "{", close: "}" },
    { open: "[", close: "]" },
    { open: '"', close: '"' },
  ],
});

export { MONGO_LANGUAGE };

// Self-hosted worker — no CDN fetch at runtime, so the console works offline.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };

export default monaco;
