import { useEffect, useMemo, useRef, useState } from "react";
import dagre from "@dagrejs/dagre";
import {
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  MarkerType,
  Panel,
  Position,
  ReactFlow,
  useReactFlow,
  type Edge,
  type Node,
  type NodeProps,
  useNodesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { LayoutGrid, Maximize2, Minimize2, RotateCcw, Scan } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export interface ErdColumn {
  name: string;
  type: string;
  nullable: boolean;
  primaryKey?: boolean;
  unique?: boolean;
  references?: { schema?: string; table?: string; column?: string };
}

export interface ErdIndex {
  name: string;
  columns: string[];
  unique?: boolean;
  primary?: boolean;
}

export interface ErdTable {
  id: string;
  schema: string;
  name: string;
  columns: ErdColumn[];
  indexes?: ErdIndex[];
  estimatedRows?: number;
  sizeBytes?: number;
}

const NODE_WIDTH = 260;
const HEADER_HEIGHT = 52;
const ROW_HEIGHT = 20;
const MAX_ROWS = 12;

type SavedLayout = Record<string, { x: number; y: number }>;

function readLayout(key?: string): SavedLayout {
  if (!key) return {};
  try {
    const value = JSON.parse(localStorage.getItem(key) ?? "{}") as SavedLayout;
    return Object.fromEntries(
      Object.entries(value).filter(([, point]) => Number.isFinite(point?.x) && Number.isFinite(point?.y)),
    );
  } catch {
    return {};
  }
}

function writeLayout(key: string | undefined, layout: SavedLayout) {
  if (!key) return;
  try {
    localStorage.setItem(key, JSON.stringify(layout));
  } catch {
    // private mode or a full quota: the graph still works, it just forgets the arrangement
  }
}

interface TableNodeData extends Record<string, unknown> {
  table: ErdTable;
  dimmed: boolean;
  sourceColumns: string[];
  targetColumns: string[];
}

type TableNodeType = Node<TableNodeData, "table">;

const nodeTypes = { table: TableNode };

function tableHeight(table: ErdTable) {
  const rows = Math.min(table.columns.length, MAX_ROWS) + (table.columns.length > MAX_ROWS ? 1 : 0);
  return HEADER_HEIGHT + rows * ROW_HEIGHT + 8;
}

function keyLabel(col: ErdColumn) {
  if (col.primaryKey) return "PK";
  if (col.references) return "FK";
  if (col.unique) return "UK";
  return "";
}

// TableNode is the readable unit of the diagram: keys and types live on the node, so the graph does not need a legend.
function TableNode({ data }: NodeProps<TableNodeType>) {
  const { table, dimmed, sourceColumns, targetColumns } = data;
  const columns = table.columns.slice(0, MAX_ROWS);
  const hidden = table.columns.length - columns.length;
  const hasSource = (column: string) => sourceColumns.includes(column);
  const hasTarget = (column: string) => targetColumns.includes(column);
  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border bg-card text-card-foreground shadow-sm transition-opacity",
        dimmed && "opacity-25",
      )}
    >
      <Handle id="target-default" type="target" position={Position.Top} className="!size-1.5 !border-0 !bg-muted-foreground" />
      <div className="border-b bg-muted/40 px-3 py-2">
        <div className="truncate text-sm font-medium">{table.name}</div>
        <div className="truncate text-[10px] text-muted-foreground">{table.schema}</div>
      </div>
      <div className="py-1">
        {columns.map((col) => (
          <div key={col.name} className="relative flex h-5 items-center gap-2 px-2 font-mono text-[11px]">
            {hasTarget(col.name) && (
              <Handle
                id={`target-${col.name}`}
                type="target"
                position={Position.Left}
                className="!size-1.5 !border-0 !bg-foreground"
              />
            )}
            <span className="w-4 shrink-0 text-[9px] font-semibold text-muted-foreground">{keyLabel(col)}</span>
            <span className={cn("truncate", col.primaryKey && "font-semibold")}>{col.name}</span>
            <span className="ml-auto max-w-[45%] truncate text-[10px] text-muted-foreground">{col.type}</span>
            {hasSource(col.name) && (
              <Handle
                id={`source-${col.name}`}
                type="source"
                position={Position.Right}
                className="!size-1.5 !border-0 !bg-foreground"
              />
            )}
          </div>
        ))}
        {hidden > 0 && (
          <div className="px-2 py-0.5 text-[10px] text-muted-foreground">+{hidden} more column(s)</div>
        )}
      </div>
      <Handle id="source-default" type="source" position={Position.Bottom} className="!size-1.5 !border-0 !bg-muted-foreground" />
    </div>
  );
}

function refKey(schema = "", table = "") {
  return `${schema}.${table}`;
}

// layoutTables turns the declared references into positioned nodes; only declared keys become edges.
function layoutTables(tables: ErdTable[]) {
  const graph = new dagre.graphlib.Graph();
  graph.setGraph({ rankdir: "TB", nodesep: 48, ranksep: 96, marginx: 32, marginy: 32 });
  graph.setDefaultEdgeLabel(() => ({}));

  const byKey = new Map<string, ErdTable>();
  const byName = new Map<string, ErdTable>();
  for (const table of tables) {
    graph.setNode(table.id, { width: NODE_WIDTH, height: tableHeight(table) });
    byKey.set(refKey(table.schema, table.name), table);
    byName.set(table.name, table);
  }

  const sourceByTable = new Map<string, Set<string>>();
  const targetByTable = new Map<string, Set<string>>();
  const visibleColumns = (table: ErdTable) => new Set(table.columns.slice(0, MAX_ROWS).map((col) => col.name));
  const edges: Edge[] = [];
  for (const table of tables) {
    for (const col of table.columns) {
      const ref = col.references;
      if (!ref?.table) continue;
      const parent = (ref.schema ? byKey.get(refKey(ref.schema, ref.table)) : undefined) ?? byName.get(ref.table);
      if (!parent) continue;
      if (parent.id !== table.id) graph.setEdge(parent.id, table.id);
      const visible = visibleColumns(table);
      const parentVisible = visibleColumns(parent);
      const sourceColumn = ref.column && parentVisible.has(ref.column) ? ref.column : undefined;
      const targetColumn = visible.has(col.name) ? col.name : undefined;
      if (sourceColumn) {
        const set = sourceByTable.get(parent.id) ?? new Set();
        set.add(sourceColumn);
        sourceByTable.set(parent.id, set);
      }
      if (targetColumn) {
        const set = targetByTable.get(table.id) ?? new Set();
        set.add(targetColumn);
        targetByTable.set(table.id, set);
      }
      edges.push({
        id: `${table.id}.${col.name}`,
        source: parent.id,
        target: table.id,
        sourceHandle: sourceColumn ? `source-${sourceColumn}` : "source-default",
        targetHandle: targetColumn ? `target-${targetColumn}` : "target-default",
        label: col.name,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed, color: "var(--foreground)" },
        style: { stroke: "var(--foreground)", strokeWidth: 1.5 },
        labelStyle: { fill: "var(--muted-foreground)", fontSize: 10 },
        labelBgStyle: { fill: "var(--background)", fillOpacity: 0.85 },
      });
    }
  }

  dagre.layout(graph);
  const nodes: Node<TableNodeData>[] = tables.map((table) => {
    const point = graph.node(table.id) as { x: number; y: number } | undefined;
    return {
      id: table.id,
      type: "table",
      position: { x: (point?.x ?? 0) - NODE_WIDTH / 2, y: (point?.y ?? 0) - tableHeight(table) / 2 },
      width: NODE_WIDTH,
      height: tableHeight(table),
      data: {
        table,
        dimmed: false,
        sourceColumns: [...(sourceByTable.get(table.id) ?? [])],
        targetColumns: [...(targetByTable.get(table.id) ?? [])],
      },
    };
  });
  return { nodes, edges };
}

function FocusSelection({ id }: { id?: string }) {
  const { getNode, setCenter } = useReactFlow();
  useEffect(() => {
    if (!id) return;
    const node = getNode(id);
    if (!node) return;
    const width = node.measured?.width ?? NODE_WIDTH;
    const height = node.measured?.height ?? HEADER_HEIGHT;
    void setCenter(node.position.x + width / 2, node.position.y + height / 2, { zoom: 1.2, duration: 350 });
  }, [id, getNode, setCenter]);
  return null;
}

function GraphActions({
  fullscreen,
  onFullscreen,
  onArrange,
}: {
  fullscreen: boolean;
  onFullscreen: () => void;
  onArrange: () => void;
}) {
  const { fitView, zoomTo } = useReactFlow();
  return (
    <Panel position="top-right" className="flex gap-1">
      <Button size="icon-sm" variant="outline" title="Fit diagram" onClick={() => void fitView({ duration: 300, padding: 0.2 })}>
        <Scan />
      </Button>
      <Button
        size="icon-sm"
        variant="outline"
        title="Auto arrange"
        aria-label="Auto arrange"
        onClick={() => {
          onArrange();
          requestAnimationFrame(() => void fitView({ duration: 300, padding: 0.2 }));
        }}
      >
        <LayoutGrid />
      </Button>
      <Button size="icon-sm" variant="outline" title="Actual size" onClick={() => void zoomTo(1, { duration: 200 })}>
        <RotateCcw />
      </Button>
      <Button
        size="icon-sm"
        variant="outline"
        title={fullscreen ? "Exit fullscreen" : "Fullscreen"}
        aria-label={fullscreen ? "Exit fullscreen" : "Fullscreen"}
        onClick={onFullscreen}
      >
        {fullscreen ? <Minimize2 /> : <Maximize2 />}
      </Button>
    </Panel>
  );
}

export function ErdCanvas({
  tables,
  selectedId,
  query,
  layoutKey,
  onSelect,
}: {
  tables: ErdTable[];
  selectedId?: string;
  query: string;
  layoutKey?: string;
  onSelect: (id: string) => void;
}) {
  const graph = useMemo(() => layoutTables(tables), [tables]);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<TableNodeData>>([]);
  const storageKey = layoutKey ? `noxis.erd.layout:${layoutKey}` : undefined;
  const savedLayout = useRef<SavedLayout>(readLayout(storageKey));
  const containerRef = useRef<HTMLDivElement>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const [colorMode, setColorMode] = useState<"dark" | "light">(() =>
    document.documentElement.classList.contains("dark") ? "dark" : "light",
  );
  const needle = query.trim().toLowerCase();

  useEffect(() => {
    const root = document.documentElement;
    const sync = () => setColorMode(root.classList.contains("dark") ? "dark" : "light");
    const observer = new MutationObserver(sync);
    observer.observe(root, { attributes: true, attributeFilter: ["class"] });
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    const onChange = () => setFullscreen(document.fullscreenElement === containerRef.current);
    document.addEventListener("fullscreenchange", onChange);
    return () => document.removeEventListener("fullscreenchange", onChange);
  }, []);

  async function toggleFullscreen() {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await containerRef.current?.requestFullscreen();
    } catch {
      // browsers may refuse fullscreen; the graph stays usable in place
    }
  }

  // A new schema replaces the layout, but selection and filtering must not undo a drag.
  useEffect(() => {
    setNodes(
      graph.nodes.map((node) => {
        const saved = savedLayout.current[node.id];
        return saved ? { ...node, position: saved } : node;
      }),
    );
  }, [graph.nodes, setNodes]);

  const viewNodes = useMemo(
    () =>
      nodes.map((node) => ({
        ...node,
        selected: node.id === selectedId,
        data: {
          ...node.data,
          dimmed: needle.length > 0 && !`${node.data.table.schema}.${node.data.table.name}`.toLowerCase().includes(needle),
        },
      })),
    [nodes, selectedId, needle],
  );

  return (
    <div ref={containerRef} className="h-full w-full bg-background">
      <ReactFlow
        nodes={viewNodes}
      edges={graph.edges}
      nodeTypes={nodeTypes}
      onNodesChange={onNodesChange}
      onNodeDragStop={(_, node) => {
        savedLayout.current[node.id] = { x: node.position.x, y: node.position.y };
        writeLayout(storageKey, savedLayout.current);
      }}
      colorMode={colorMode}
      fitView
      fitViewOptions={{ padding: 0.2 }}
      minZoom={0.1}
      maxZoom={2}
      nodesConnectable={false}
      onNodeClick={(_, node) => onSelect(node.id)}
        className="bg-muted/20"
      >
        <Background variant={BackgroundVariant.Dots} gap={18} size={1} />
        <Controls showInteractive={false} />
        <GraphActions
          fullscreen={fullscreen}
          onFullscreen={() => void toggleFullscreen()}
          onArrange={() => {
            savedLayout.current = {};
            writeLayout(storageKey, {});
            setNodes(graph.nodes);
          }}
        />
        <FocusSelection id={selectedId} />
        <FitOnLoad count={nodes.length} />
      </ReactFlow>
    </div>
  );
}

// FitOnLoad waits for measured nodes, which the initial fitView cannot do while the schema is still loading.
function FitOnLoad({ count }: { count: number }) {
  const { fitView } = useReactFlow();
  useEffect(() => {
    if (count > 0) void fitView({ duration: 200, padding: 0.2 });
  }, [count, fitView]);
  return null;
}
