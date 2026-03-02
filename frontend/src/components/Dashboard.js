import React, { useState, useEffect } from 'react';
import { 
  LineChart, Line, AreaChart, Area, XAxis, YAxis, 
  CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell 
} from 'recharts';
import { Cpu, HardDrive, Network, Activity, AlertTriangle } from 'lucide-react';
import axios from 'axios';

const Dashboard = ({ metrics }) => {
  const [modules, setModules] = useState([]);
  const [activeModules, setActiveModules] = useState(0);
  const [history, setHistory] = useState([]);

  useEffect(() => {
    fetchModules();
    const interval = setInterval(fetchModules, 2000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (metrics) {
      setHistory(prev => {
        const newHistory = [...prev, {
          time: new Date().toLocaleTimeString(),
          cpu: metrics.cpu.usage_percent,
          memory: metrics.memory.usage_percent,
          disk: metrics.disk.usage_percent
        }];
        return newHistory.slice(-20);
      });
    }
  }, [metrics]);

  const fetchModules = async () => {
    try {
      const response = await axios.get('/api/v1/modules');
      setModules(response.data.modules);
      let active = 0;
      for (const mod of response.data.modules) {
        const statusRes = await axios.get(`/api/v1/modules/${mod.id}/status`);
        if (statusRes.data.running) active++;
      }
      setActiveModules(active);
    } catch (error) {
      console.error('Failed to fetch modules:', error);
    }
  };

  const StatCard = ({ title, value, icon: Icon, color, subtitle }) => (
    <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-slate-400 text-sm">{title}</p>
          <p className={`text-3xl font-bold mt-1 ${color}`}>{value}</p>
          {subtitle && <p className="text-slate-500 text-xs mt-1">{subtitle}</p>}
        </div>
        <div className={`p-3 rounded-lg bg-slate-700/50`}>
          <Icon className={color} size={24} />
        </div>
      </div>
    </div>
  );

  const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899'];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-white">Dashboard</h2>
        {activeModules > 0 && (
          <div className="flex items-center space-x-2 text-amber-400 animate-pulse">
            <AlertTriangle size={20} />
            <span>{activeModules} модулей активно</span>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="CPU Usage"
          value={metrics ? `${metrics.cpu.usage_percent.toFixed(1)}%` : '--'}
          icon={Cpu}
          color={metrics?.cpu.usage_percent > 80 ? 'text-red-400' : 'text-blue-400'}
          subtitle={`${metrics?.cpu.core_count || '--'} cores`}
        />
        <StatCard
          title="Memory Usage"
          value={metrics ? `${metrics.memory.usage_percent.toFixed(1)}%` : '--'}
          icon={Activity}
          color={metrics?.memory.usage_percent > 80 ? 'text-red-400' : 'text-green-400'}
          subtitle={metrics ? `${(metrics.memory.used / 1024 / 1024 / 1024).toFixed(1)} GB` : '--'}
        />
        <StatCard
          title="Disk Usage"
          value={metrics ? `${metrics.disk.usage_percent.toFixed(1)}%` : '--'}
          icon={HardDrive}
          color="text-amber-400"
          subtitle={metrics ? `${(metrics.disk.free / 1024 / 1024 / 1024).toFixed(1)} GB free` : '--'}
        />
        <StatCard
          title="Active Modules"
          value={activeModules}
          icon={Network}
          color={activeModules > 0 ? 'text-red-400' : 'text-slate-400'}
          subtitle={`${modules.length} available`}
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">Real-time System Load</h3>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={history}>
              <CartesianGrid strokeDasharray="3 3" stroke="#334155" />
              <XAxis dataKey="time" stroke="#64748b" fontSize={12} />
              <YAxis stroke="#64748b" fontSize={12} unit="%" />
              <Tooltip 
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155' }}
                labelStyle={{ color: '#94a3b8' }}
              />
              <Area 
                type="monotone" 
                dataKey="cpu" 
                stackId="1" 
                stroke="#3b82f6" 
                fill="#3b82f6" 
                fillOpacity={0.3}
                name="CPU %"
              />
              <Area 
                type="monotone" 
                dataKey="memory" 
                stackId="1" 
                stroke="#10b981" 
                fill="#10b981" 
                fillOpacity={0.3}
                name="Memory %"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">Module Categories</h3>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={[
                  { name: 'CPU', value: modules.filter(m => m.category === 'cpu').length },
                  { name: 'Memory', value: modules.filter(m => m.category === 'memory').length },
                  { name: 'Disk', value: modules.filter(m => m.category === 'disk').length },
                  { name: 'Network', value: modules.filter(m => m.category === 'network').length },
                  { name: 'App', value: modules.filter(m => m.category === 'application').length },
                  { name: 'Security', value: modules.filter(m => m.category === 'security').length },
                ].filter(d => d.value > 0)}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={100}
                paddingAngle={5}
                dataKey="value"
              >
                {modules.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip 
                contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155' }}
              />
            </PieChart>
          </ResponsiveContainer>
          <div className="flex flex-wrap gap-2 mt-4 justify-center">
            {['CPU', 'Memory', 'Disk', 'Network', 'App', 'Security'].map((cat, idx) => (
              <div key={cat} className="flex items-center space-x-1">
                <div 
                  className="w-3 h-3 rounded-full" 
                  style={{ backgroundColor: COLORS[idx % COLORS.length] }}
                />
                <span className="text-xs text-slate-400">{cat}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
        <h3 className="text-lg font-semibold text-white mb-4">Target Configuration</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-slate-700/50 rounded p-4">
            <p className="text-slate-400 text-sm">IP Address</p>
            <p className="text-xl font-mono text-blue-400">185.40.76.46</p>
          </div>
          <div className="bg-slate-700/50 rounded p-4">
            <p className="text-slate-400 text-sm">Domain</p>
            <p className="text-xl font-mono text-green-400">mvp2001.ru</p>
          </div>
          <div className="bg-slate-700/50 rounded p-4">
            <p className="text-slate-400 text-sm">Status</p>
            <div className="flex items-center space-x-2">
              <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
              <span className="text-green-400">Ready</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
