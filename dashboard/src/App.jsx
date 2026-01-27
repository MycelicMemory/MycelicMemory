import { Routes, Route, NavLink } from 'react-router-dom';
import { Brain, Search, Settings } from 'lucide-react';
import Overview from './pages/Overview';
import MemoryBrowser from './pages/MemoryBrowser';

function App() {
  return (
    <div className="min-h-screen flex">
      {/* Sidebar */}
      <aside className="w-64 bg-slate-800 border-r border-slate-700 flex flex-col">
        {/* Logo */}
        <div className="p-6 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 bg-primary-500 rounded-xl flex items-center justify-center">
              <Brain className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="font-semibold text-lg">Ultrathink</h1>
              <p className="text-xs text-slate-400">Memory Dashboard</p>
            </div>
          </div>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-4">
          <ul className="space-y-2">
            <li>
              <NavLink
                to="/"
                className={({ isActive }) =>
                  `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-primary-500/20 text-primary-400'
                      : 'text-slate-400 hover:bg-slate-700 hover:text-slate-200'
                  }`
                }
              >
                <Brain className="w-5 h-5" />
                Overview
              </NavLink>
            </li>
            <li>
              <NavLink
                to="/memories"
                className={({ isActive }) =>
                  `flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
                    isActive
                      ? 'bg-primary-500/20 text-primary-400'
                      : 'text-slate-400 hover:bg-slate-700 hover:text-slate-200'
                  }`
                }
              >
                <Search className="w-5 h-5" />
                Memory Browser
              </NavLink>
            </li>
          </ul>
        </nav>

        {/* Status */}
        <div className="p-4 border-t border-slate-700">
          <div className="flex items-center gap-2 text-sm">
            <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            <span className="text-slate-400">Connected to API</span>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-auto">
        <Routes>
          <Route path="/" element={<Overview />} />
          <Route path="/memories" element={<MemoryBrowser />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
