import React, { useState, useEffect } from "react";
import api from "./api";
import { CircleCheckBig, LogOut } from "lucide-react";

export default function Dashboard({ onLogout }) {

  const [data, setData] = useState(null);
  const [showSuccess, setShowSuccess] = useState(false);
  const [completedTasks, setCompletedTasks] = useState(new Set());

  const loadData = async () => {
    try {
      const res = await api.get(`/daily-check-in`);
      setData(res.data);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const completeTask = async (habitId, minutes) => {
    try {
      await api.post(`/complete-task`, {
        habitId,
        minutes
      });

      setCompletedTasks(prev => new Set(prev).add(habitId));
      setShowSuccess(true);
      setTimeout(() => setShowSuccess(false), 2000);

      loadData();

    } catch (err) {
      console.error(err);
    }
  };

  if (!data)
    return (
      <div className="p-20 text-center text-indigo-600 font-bold">
        Syncing your discipline...
      </div>
    );

  return (
    <div className="max-w-xl mx-auto p-6 pt-12">

      {showSuccess && (
        <div className="fixed top-5 right-5 bg-green-500 text-white p-4 rounded-lg shadow-lg">
          Task completed successfully!
        </div>
      )}

      <div className="bg-indigo-600 rounded-3xl p-10 text-white mb-10 text-center">
        <div className="text-8xl mb-4">
          {data.lotusStatus === "seedling"
            ? "🌱"
            : data.lotusStatus === "sprout"
            ? "🌿"
            : data.lotusStatus === "bud"
            ? "🌷"
            : "🪷"}
        </div>

        <h2 className="text-2xl font-bold">{data.phase} Phase</h2>

        <p>Day {data.dayInProgram} of 66</p>

        <div className="mt-6 bg-white/20 h-2 rounded-full">
          <div
            className="bg-white h-full"
            style={{ width: `${data.growthPercent}%` }}
          />
        </div>
      </div>

      <div className="space-y-4">
        {data.habits.map((h) => (
          <div
            key={h.id}
            className={`p-6 bg-white rounded-2xl border flex justify-between items-center ${
              completedTasks.has(h.id) || h.completed ? "bg-green-100" : ""
            }`}
          >
            <div>
              <p className="font-bold text-xl">{h.name}</p>
              <p className="text-sm text-slate-400">
                {h.currentMinutes} {h.unit} required
              </p>
            </div>

            <button
              onClick={() => completeTask(h.id, h.currentMinutes)}
              disabled={completedTasks.has(h.id) || h.completed}
              className="p-4 rounded-xl hover:bg-indigo-600 hover:text-white disabled:bg-slate-300"
            >
              <CircleCheckBig size={30} />
            </button>
          </div>
        ))}
      </div>

      <button
        onClick={onLogout}
        className="mt-10 text-red-500 flex items-center gap-2"
      >
        <LogOut size={18} /> Sign Out
      </button>

    </div>
  );
}