export interface FileTreeNode {
  name: string;
  path: string;
  type: "file" | "directory";
  children?: FileTreeNode[];
  isExpanded?: boolean;
  isDirty?: boolean; // Added isDirty
}

interface FlatFile {
  path: string;
  isDirty: boolean;
}

export function buildFileTree(flatFiles: FlatFile[]): FileTreeNode[] {
  const tree: FileTreeNode[] = [];

  flatFiles.forEach((flatFile) => {
    const fullPath = flatFile.path;
    const parts = fullPath.split("/");
    let currentLevel = tree;
    let currentPath = "";

    parts.forEach((part, index) => {
      currentPath = currentPath === "" ? part : `${currentPath}/${part}`;
      const existingNode = currentLevel.find((node) => node.name === part);

      if (existingNode) {
        if (existingNode.type === "directory") {
          currentLevel = existingNode.children!;
        }
        // If it's a file and already exists, update its dirty status
        if (existingNode.type === "file" && index === parts.length - 1) {
          existingNode.isDirty = flatFile.isDirty;
        }
      } else {
        const newNode: FileTreeNode = {
          name: part,
          path: currentPath,
          type: index === parts.length - 1 ? "file" : "directory",
          isExpanded: false,
          isDirty: index === parts.length - 1 ? flatFile.isDirty : undefined, // Only files can be dirty
        };

        if (newNode.type === "directory") {
          newNode.children = [];
        }

        currentLevel.push(newNode);

        if (newNode.type === "directory") {
          currentLevel = newNode.children!;
        }
      }
    });
  });

  // Propagate dirty state up the tree
  const propagateDirtyState = (nodes: FileTreeNode[]) => {
    nodes.forEach((node) => {
      if (node.type === "directory" && node.children) {
        propagateDirtyState(node.children);
        node.isDirty = node.children.some((child) => child.isDirty);
      }
    });
  };
  propagateDirtyState(tree);

  // Sort directories and files
  const sortTree = (nodes: FileTreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.type === "directory" && b.type === "file") return -1;
      if (a.type === "file" && b.type === "directory") return 1;
      return a.name.localeCompare(b.name);
    });
    nodes.forEach((node) => {
      if (node.children) {
        sortTree(node.children);
      }
    });
  };

  sortTree(tree);
  return tree;
}
