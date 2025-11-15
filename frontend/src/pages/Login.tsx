import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useLoginMutation } from '../hooks/useLoginMutation';

export default function Login() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const { login, isInFlight } = useLoginMutation({
    onCompleted: (response, errors) => {
      if (errors) {
        setError(errors[0].message);
        return;
      }
      if (response.login?.token) {
        localStorage.setItem('token', response.login.token);
        navigate('/projects');
      } else {
        setError('Login failed: No token received.');
      }
    },
    onError: (err) => {
      setError('Failed to login. Please try again.');
      console.error(err);
    },
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    login(email, password);
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-base-200">
      <div className="card w-96 bg-base-100 shadow-xl">
        <div className="card-body">
          <h2 className="card-title text-2xl font-bold text-center">LaTeX Editor</h2>
          <p className="text-center text-base-content/70 mb-4">Sign in to your account</p>

          {error && (
            <div className="alert alert-error">
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit}>
            <div className="form-control">
              <label className="label">
                <span className="label-text">Email</span>
              </label>
              <input
                type="email"
                placeholder="email@example.com"
                className="input input-bordered"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>

            <div className="form-control mt-4">
              <label className="label">
                <span className="label-text">Password</span>
              </label>
              <input
                type="password"
                placeholder="••••••••"
                className="input input-bordered"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            <div className="form-control mt-6">
              <button
                type="submit"
                className={`btn btn-primary ${isInFlight ? 'loading' : ''}`}
                disabled={isInFlight}
              >
                {isInFlight ? 'Signing in...' : 'Sign In'}
              </button>
            </div>
          </form>

          <div className="divider">OR</div>

          <p className="text-center">
            Don't have an account?{' '}
            <Link to="/register" className="link link-primary">
              Sign up
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
