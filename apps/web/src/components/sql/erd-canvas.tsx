import { useEffect, useMemo, useRef, useState } from "react";
import dagre from "@dagrejs/dagre";
import {
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
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
import { Maximize2, Minimize2, RotateCcw, Scan } from "lucide-react";
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

interface TableNodeData extends Record<string, unknown> {
  table: ErdTable;
  dimmed: boolean;
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
  const { table, dimmed } = data;
  const columns = table.columns.slice(0, MAX_ROWS);
  const hidden = table.columns.length - columns.length;
  return (
    <div
      className={cn(
        "overflow-hidden rounded-lg border bg-card text-card-foreground shadow-sm transition-opacity",
        dimmed && "opacity-25",
      )}
    >
      <Handle type="target" position={Position.Top} className="!size-1.5 !border-0 !bg-muted-foreground" />
      <div className="border-b bg-muted/40 px-3 py-2">
        <div className="truncate text-sm font-medium">{table.name}</div>
        <div className="truncate text-[10px] text-muted-foreground">{table.schema}</div>
      </div>
      <div className="py-1">
        {columns.map((col) => (
          <div key={col.name} className="flex h-5 items-center gap-2 px-2 font-mono text-[11px]">
            <span className="w-4 shrink-0 text-[9px] font-semibold text-muted-foreground">{keyLabel(col)}</span>
            <span className={cn("truncate", col.primaryKey && "font-semibold")}>{col.name}</span>
            <span className="ml-auto max-w-[45%] truncate text-[10px] text-muted-foreground">{col.type}</span>
          </div>
        ))}
        {hidden > 0 && (
          <div className="px-2 py-0.5 text-[10px] text-muted-foreground">+{hidden} more column(s)</div>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!size-1.5 !border-0 !bg-muted-foreground" />
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

  const edges: Edge[] = [];
  for (const table of tables) {
    for (const col of table.columns) {
      const ref = col.references;
      if (!ref?.table) continue;
      const parent = (ref.schema ? byKey.get(refKey(ref.schema, ref.table)) : undefined) ?? byName.get(ref.table);
      if (!parent) continue;
      if (parent.id !== table.id) graph.setEdge(parent.id, table.id);
      edges.push({
        id: `${table.id}.${col.name}`,
        source: parent.id,
        target: table.id,
        label: col.name,
        type: "smoothstep",
        markerEnd: { type: MarkerType.ArrowClosed, color: "var(--muted-foreground)" },
        style: { stroke: "var(--border)" },
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
      data: { table, dimmed: false },
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

function GraphActions({ fullscreen, onFullscreen }: { fullscreen: boolean; onFullscreen: () => void }) {
  const { fitView, zoomTo } = useReactFlow();
  return (
    <Panel position="top-right" className="flex gap-1">
      <Button size="icon-sm" variant="outline" title="Fit diagram" onClick={() => void fitView({ duration: 300, padding: 0.2 })}>
        <Scan />
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
  onSelect,
}: {
  tables: ErdTable[];
  selectedId?: string;
  query: string;
  onSelect: (id: string) => void;
}) {
  const graph = useMemo(() => layoutTables(tables), [tables]);
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<TableNodeData>>([]);
  const containerRef = useRef<HTMLDivElement>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const needle = query.trim().toLowerCase();

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
    setNodes(graph.nodes);
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
      colorMode="dark"
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
        <MiniMap pannable zoomable nodeColor="#94a3b8" maskColor="rgba(2, 6, 23, 0.72)" />
        <GraphActions fullscreen={fullscreen} onFullscreen={() => void toggleFullscreen()} />
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
