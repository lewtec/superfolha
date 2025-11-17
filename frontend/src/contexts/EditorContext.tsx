import React, { createContext, useContext } from "react";

interface EditorContextType {
  projectName: string;
  onCompile: () => void;
  compiling: boolean;
  editorStatus: "clean" | "dirty" | "saving" | "saved" | "error";
  // Add other editor-specific state/functions if needed in the layout
}

const EditorContext = createContext<EditorContextType | undefined>(undefined);

export const EditorProvider: React.FC<{
  children: React.ReactNode;
  value: EditorContextType;
}> = ({ children, value }) => {
  return (
    <EditorContext.Provider value={value}>{children}</EditorContext.Provider>
  );
};

export const useEditor = () => {
  const context = useContext(EditorContext);
  // No need to throw error here, as Layout will check for undefined
  return context;
};
