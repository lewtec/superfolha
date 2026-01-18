interface NavbarProps {
  projectName: string;
  onCompile: () => void;
  compiling: boolean;
}

export default function Navbar({
  projectName,
  onCompile,
  compiling,
}: NavbarProps) {
  // const navigate = useNavigate() // navigate is not used anymore

  return (
    <div className="navbar bg-base-100 shadow-md">
      <div className="navbar-start">
        <span className="ml-4 text-base-content/70">
          Project: {projectName}
        </span>
      </div>
      <div className="navbar-end gap-2">
        <button
          className={`
            btn btn-primary
            ${compiling ? "loading" : ""}
          `}
          onClick={onCompile}
          disabled={compiling}
        >
          {compiling ? "Compiling..." : "Compile"}
        </button>
      </div>
    </div>
  );
}
