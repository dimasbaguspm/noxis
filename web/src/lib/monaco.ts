import * as monaco from "monaco-editor/editor/editor.api.js";
import "monaco-editor/languages/definitions/sql/register.js";
import "monaco-editor/languages/definitions/pgsql/register.js";
import "monaco-editor/languages/definitions/mysql/register.js";
import editorWorker from "monaco-editor/editor/editor.worker.js?worker";

// Self-hosted and trimmed: no CDN fetch at runtime (works offline in a
// homelab) and only the SQL dialects are registered — the other ~90 language
// chunks never enter the bundle.
self.MonacoEnvironment = { getWorker: () => new editorWorker() };

export default monaco;
