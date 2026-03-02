import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

const Monitoring = ({ metrics }) => {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold text-white">Real-time Monitoring</h2>
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">CPU Cores</h3>
          {metrics?.cpu?.per_core_usage && (
            <div className="space-y-2">
              {metrics.cpu.per_core_usage.map((usage, idx) => (
                <div key={idx} className="flex items-center space-x-3">
                  <span className="text-slate-400 w-16">Core {idx}</span>
                  <div className="flex-1 bg-slate-700 rounded-full h-4">
                    <div 
                      className={`h-full rounded-full ${usage > 80 ? 'bg-red-500' : 'bg-blue-500'}`}
                      style={{ width: `${usage}%` }}
                    />
                  </div>
                  <span className="text-slate-300 w-12 text-right">{usage.toFixed(1)}%</span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">Load Average</h3>
          {metrics?.load_avg && (
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-slate-400">1 min</span>
                <span className="text-2xl font-mono text-blue-400">{metrics.load_avg.load1.toFixed(2)}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-slate-400">5 min</span>
                <span className="text-2xl font-mono text-green-400">{metrics.load_avg.load5.toFixed(2)}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-slate-400">15 min</span>
                <span className="text-2xl font-mono text-amber-400">{metrics.load_avg.load15.toFixed(2)}</span>
              </div>
            </div>
          )}
        </div>

        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">Network</h3>
          {metrics?.network && (
            <div className="space-y-3">
              <div className="flex justify-between">
                <span className="text-slate-400">Bytes Sent/sec</span>
                <span className="font-mono text-blue-400">
                  {(metrics.network.bytes_sent / 1024).toFixed(2)} KB
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Bytes Recv/sec</span>
                <span className="font-mono text-green-400">
                  {(metrics.network.bytes_recv / 1024).toFixed(2)} KB
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Active Connections</span>
                <span className="font-mono text-amber-400">
                  {metrics.network.connections}
                </span>
              </div>
            </div>
          )}
        </div>

        <div className="bg-slate-800 rounded-lg p-6 border border-slate-700">
          <h3 className="text-lg font-semibold text-white mb-4">Processes</h3>
          <div className="text-center">
            <span className="text-5xl font-bold text-purple-400">
              {metrics?.processes || '--'}
            </span>
            <p className="text-slate-400 mt-2">Total Processes</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Monitoring;
