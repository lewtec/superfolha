export interface FileTreeNode {
  name: string;
  path: string;
  type: 'file' | 'directory';
  children?: FileTreeNode[];
  isExpanded?: boolean;
}

export function buildFileTree(filePaths: string[]): FileTreeNode[] {
  const tree: FileTreeNode[] = [];

  filePaths.forEach(fullPath => {
    const parts = fullPath.split('/');
    let currentLevel = tree;
    let currentPath = '';

    parts.forEach((part, index) => {
      currentPath = currentPath === '' ? part : `${currentPath}/${part}`;
      const existingNode = currentLevel.find(node => node.name === part);

      if (existingNode) {
        if (existingNode.type === 'directory') {
          currentLevel = existingNode.children!;
        }
      } else {
        const newNode: FileTreeNode = {
          name: part,
          path: currentPath,
          type: index === parts.length - 1 ? 'file' : 'directory',
          isExpanded: false,
        };

        if (newNode.type === 'directory') {
          newNode.children = [];
        }

        currentLevel.push(newNode);

        if (newNode.type === 'directory') {
          currentLevel = newNode.children!;
        }
      }
    });
  });

  // Sort directories and files
  const sortTree = (nodes: FileTreeNode[]) => {
    nodes.sort((a, b) => {
      if (a.type === 'directory' && b.type === 'file') return -1;
      if (a.type === 'file' && b.type === 'directory') return 1;
      return a.name.localeCompare(b.name);
    });
    nodes.forEach(node => {
      if (node.children) {
        sortTree(node.children);
      }
    });
  };

  sortTree(tree);
  return tree;
}
