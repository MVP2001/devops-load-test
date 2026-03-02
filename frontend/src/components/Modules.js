import React, { useState, useEffect } from 'react';
import { 
  Play, Square, Settings, AlertTriangle, 
  Cpu, HardDrive, Network, Shield, Database, FileText 
} from 'lucide-react';
import axios from 'axios';

const Modules = () => {
  const [modules, setModules] = useState([]);
  const [selectedModule, setSelectedModule] = useState(null);
  const [config, setConfig] = useState({
    vm: { ip: '185.40.76.46', port: 80 },
    duration: 60000000000,
    intensity: 'medium',
    concurrent_users: 100,
    custom_params: {}
  });
  const [runningModules, setRunningModules] = useState(new Set());

  useEffect(() => {
    fetchModules();
    const interval = setInterval(fetchModules, 2000);
    return () => clearInterval(interval);
  }, []);

  const fetchModules = async () => {
    try {
      const response = await axios.get('/api/v1/modules');
      setModules(response.data.modules);
      
      const running = new Set();
      for (const mod of response.data.modules) {
        const statusRes = await axios.get(`/api/v1/modules/${mod.id}/status`);
        if (statusRes.data.running) {
          running.add(mod.id);
        }
      }
      setRunningModules(running);
    } catch (error) {
      console.error('Failed to fetch modules:', error);
    }
  };

  const startModule = async (moduleId) => {
    try {
      await axios.post(`/api/v1/modules/${moduleId}/start`, config);
      setRunningModules(prev => new Set(prev).add(moduleId));
    } catch (error) {
      alert('Failed to start module: ' + error.response?.data?.error || error.message);
    }
  };

  const stopModule = async (moduleId) => {
    try {
      await axios.post(`/api/v1/modules/${moduleId}/stop`);
      setRunningModules(prev => {
        const newSet = new Set(prev);
        newSet.delete(moduleId);
        return newSet;
      });
    } catch (error) {
      alert('Failed to stop module: ' + error.message);
    }
  };

  const getCategoryIcon = (category) => {
    switch (category) {
      case 'cpu': return Cpu;
      case 'memory': return Activity;
      case 'disk': return HardDrive;
      case 'network': return Network;
      case 'application': return Database;
      case 'security': return Shield;
      default: return FileText;
    }
  };

  const getCategoryColor = (category) => {
    switch (category) {
      case 'cpu': return 'text-red-400 bg-red-400/10';
      case 'memory': return 'text-purple-400 bg-purple-400/10';
      case 'disk': return 'text-amber-400 bg-amber-400/10';
      case 'network': return 'text-blue-400 bg-blue-400/10';
      case 'application': return 'text-green-400 bg-green-400/10';
      case 'security': return 'text-rose-400 bg-rose-400/10';
      default: return 'text-slate-400 bg-slate-400/10';
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white">Load Modules</h2>
        <div className="text-sm text-slate-400">
          {runningModules.size} active / {modules.length} total
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {modules.map((module) => {
          const Icon = getCategoryIcon(module.category);
          const isRunning = runningModules.has(module.id);
          
          return (
            <div 
              key={module.id}
              className={`
                bg-slate-800 rounded-lg p-6 border transition-all
                ${isRunning 
                  ? 'border-red-500/50 shadow-lg shadow-red-500/10' 
                  : 'border-slate-700 hover:border-slate-600'
                }
              `}
            >
              <div className="flex items-start justify-between mb-4">
                <div className={`p-3 rounded-lg ${getCategoryColor(module.category)}`}>
                  <Icon size={24} />
                </div>
                {isRunning && (
                  <div className="flex items-center space-x-1 text-red-400 animate-pulse">
                    <div className="w-2 h-2 bg-red-400 rounded-full" />
                    <span className="text-xs font-medium">RUNNING</span>
                  </div>
                )}
              </div>

              <h3 className="text-lg font-semibold text-white mb-2">{module.name}</h3>
              <p className="text-slate-400 text-sm mb-4">{module.description}</p>

              <div className="flex items-center justify-between">
                <span className={`
                  px-2 py-1 rounded text-xs font-medium uppercase
                  ${getCategoryColor(module.category)}
                `}>
                  {module.category}
                </span>
                
                <div className="flex space-x-2">
                  <button
                    onClick={() => setSelectedModule(module)}
                    className="p-2 rounded-lg bg-slate-700 text-slate-300 hover:bg-slate-600 transition-colors"
                    title="Configure"
                  >
                    <Settings size={18} />
                  </button>
                  
                  {isRunning ? (
                    <button
                      onClick={() => stopModule(module.id)}
                      className="p-2 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors"
                      title="Stop"
                    >
                      <Square size={18} />
                    </button>
                  ) : (
                    <button
                      onClick={() => startModule(module.id)}
                      className="p-2 rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 transition-colors"
                      title="Start"
                    >
                      <Play size={18} />
                    </button>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {selectedModule && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-lg max-w-2xl w-full max-h-[90vh] overflow-auto border border-slate-700">
            <div className="p-6 border-b border-slate-700">
              <div className="flex items-center justify-between">
                <h3 className="text-xl font-bold text-white">
                  Configure {selectedModule.name}
                </h3>
                <button 
                  onClick={() => setSelectedModule(null)}
                  className="text-slate-400 hover:text-white"
                >
                  ✕
                </button>
              </div>
            </div>
            
            <div className="p-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Target IP</label>
                  <input
                    type="text"
                    value={config.vm.ip}
                    onChange={(e) => setConfig({...config, vm: {...config.vm, ip: e.target.value}})}
                    className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-white"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Port</label>
                  <input
                    type="number"
                    value={config.vm.port}
                    onChange={(e) => setConfig({...config, vm: {...config.vm, port: parseInt(e.target.value)}})}
                    className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-white"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm text-slate-400 mb-1">Duration (seconds)</label>
                <input
                  type="number"
                  value={config.duration / 1000000000}
                  onChange={(e) => setConfig({...config, duration: parseInt(e.target.value) * 1000000000})}
                  className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-white"
                />
              </div>

              {selectedModule.params && selectedModule.params.map((param) => (
                <div key={param.name}>
                  <label className="block text-sm text-slate-400 mb-1">
                    {param.label}
                    {param.required && <span className="text-red-400">*</span>}
                  </label>
                  {param.type === 'select' ? (
                    <select
                      value={config.custom_params[param.name] || param.default}
                      onChange={(e) => setConfig({
                        ...config, 
                        custom_params: {...config.custom_params, [param.name]: e.target.value}
                      })}
                      className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-white"
                    >
                      {param.options.map(opt => (
                        <option key={opt} value={opt}>{opt}</option>
                      ))}
                    </select>
                  ) : param.type === 'boolean' ? (
                    <input
                      type="checkbox"
                      checked={config.custom_params[param.name] || param.default}
                      onChange={(e) => setConfig({
                        ...config,
                        custom_params: {...config.custom_params, [param.name]: e.target.checked}
                      })}
                      className="w-4 h-4 rounded bg-slate-700 border-slate-600"
                    />
                  ) : (
                    <input
                      type={param.type === 'number' ? 'number' : 'text'}
                      value={config.custom_params[param.name] || param.default}
                      onChange={(e) => setConfig({
                        ...config,
                        custom_params: {...config.custom_params, [param.name]: 
                          param.type === 'number' ? parseFloat(e.target.value) : e.target.value
                        }
                      })}
                      className="w-full bg-slate-700 border border-slate-600 rounded px-3 py-2 text-white"
                    />
                  )}
                  {param.description && (
                    <p className="text-xs text-slate-500 mt-1">{param.description}</p>
                  )}
                </div>
              ))}
            </div>

            <div className="p-6 border-t border-slate-700 flex justify-end space-x-3">
              <button
                onClick={() => setSelectedModule(null)}
                className="px-4 py-2 rounded-lg bg-slate-700 text-slate-300 hover:bg-slate-600"
              >
                Cancel
              </button>
              <button
                onClick={() => {
                  startModule(selectedModule.id);
                  setSelectedModule(null);
                }}
                className="px-4 py-2 rounded-lg bg-blue-500 text-white hover:bg-blue-600 flex items-center space-x-2"
              >
                <Play size={18} />
                <span>Start Module</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default Modules;
