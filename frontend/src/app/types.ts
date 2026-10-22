export type Menu = {
  id: string;
  name: string;
  parent_id: string | null;
  depth: number;
  position: number;
  children: Menu[];
};
