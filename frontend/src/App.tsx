import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Login from "./pages/Login";
import Register from "./pages/Register";
import Projects from "./pages/Projects";
import EditorPage from "./pages/EditorPage";
import Landing from "./pages/Landing";

import Layout from "./components/Layout";
import { useAuthStatus } from "./hooks/useAuthStatus"; // Import the new hook

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/projects"
          element={
            <ProtectedRoute>
              <Layout>
                <Projects />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/editor/:id"
          element={
            <ProtectedRoute>
              <Layout>
                <EditorPage />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route path="/" element={<Landing />} />
      </Routes>
    </BrowserRouter>
  );
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, loading } = useAuthStatus(); // Use the new hook

  if (loading) {
    // Show a loading indicator while authentication status is being determined
    return (
      <div className="flex h-screen items-center justify-center">
        <span className="loading loading-lg loading-spinner"></span>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}

export default App;
