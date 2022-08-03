type NodeData = {
  id: string;
  [key: string]: string | number;
};

export type Node = {
  data: NodeData;
};

type EdgeData = {
  id: string;
  source: Node['data']['id'];
  target: Node['data']['id'];
  [key: string]: string | number;
};

export type Edge = {
  data: EdgeData;
};

export interface CallgraphData {
  nodes: Node[];
  edges: Edge[];
}
