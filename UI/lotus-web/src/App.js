import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

const API_BASE = 'http://localhost:8080/api';

function App() {
  const [token, setToken] = useState(localStorage.getItem('lotus_token'));
  const [view, setView] = useState(token ? 'dashboard' : 'register'); // register, habits, login, dashboard
  
  // Registration State
  const [regData, setRegData] = useState({ username: '', password: '', confirmPassword: '' });
  const [habits, setHabits] = useState([
    { name: 'Lotus Sit', goalMinutes: 20, unit: 'minutes' }
  ]);

  const handleLogout = () => {
    setToken(null);
    localStorage.removeItem('lotus_token');
    setView('login');
    setRegData({ username: '', password: '', confirmPassword: '' });
  };

  const handleLoginSuccess = (newToken) => {
    setToken(newToken);
    localStorage.setItem('lotus_token', newToken);
    setView('dashboard');
  };

  return (
    <div className="app-container">
      <header className="app-header">
        <h1>Lotus Discipline</h1>
        {token && <button onClick={handleLogout} className="logout-btn">Logout</button>}
      </header>
      <main>
        {view === 'register' && (
          <RegisterStep1 
            data={regData} 
            setData={setRegData} 
            onNext={() => setView('habits')} 
            onSwitchToLogin={() => setView('login')} 
          />
        )}
        {view === 'habits' && (
          <RegisterStep2Habits 
            habits={habits} 
            setHabits={setHabits} 
            regData={regData}
            onSuccess={handleLoginSuccess}
            onBack={() => setView('register')}
          />
        )}
        {view === 'login' && (
          <Login 
            onSuccess={handleLoginSuccess} 
            onSwitchToRegister={() => setView('register')} 
          />
        )}
        {view === 'dashboard' && token && (
          <Dashboard token={token} />
        )}
      </main>
    </div>
  );
}

function RegisterStep1({ data, setData, onNext, onSwitchToLogin }) {
  const [error, setError] = useState('');

  const handleChange = (e) => {
    setData({ ...data, [e.target.name]: e.target.value });
    setError('');
  };
  
  const handleSubmit = (e) => {
    e.preventDefault();
    if (data.password !== data.confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    if (data.username && data.password) onNext();
  };

  return (
    <div className="card">
      <h2>Begin Your Journey</h2>
      <form onSubmit={handleSubmit}>
        <input name="username" placeholder="Username" value={data.username} onChange={handleChange} required />
        <input name="password" type="password" placeholder="Password" value={data.password} onChange={handleChange} required />
        <input name="confirmPassword" type="password" placeholder="Confirm Password" value={data.confirmPassword} onChange={handleChange} required />
        {error && <p className="error">{error}</p>}
        <button type="submit" className="primary-btn">Next: Define Habits</button>
      </form>
      <p className="switch-link" onClick={onSwitchToLogin}>Already have an account? Login</p>
    </div>
  );
}

function RegisterStep2Habits({ habits, setHabits, regData, onSuccess, onBack }) {
  const [newHabit, setNewHabit] = useState({ name: '', goalMinutes: 10, unit: 'minutes' });
  const [error, setError] = useState('');

  const addHabit = () => {
    if (!newHabit.name) return;
    setHabits([...habits, newHabit]);
    setNewHabit({ name: '', goalMinutes: 10, unit: 'minutes' });
  };

  const removeHabit = (idx) => {
    setHabits(habits.filter((_, i) => i !== idx));
  };

  const handleRegister = async () => {
    try {
      const { confirmPassword, ...apiData } = regData;
      const res = await axios.post(`${API_BASE}/register`, {
        ...apiData,
        habits: habits
      });
      onSuccess(res.data.token);
    } catch (err) {
      setError(err.response?.data?.error || 'Registration failed');
    }
  };

  return (
    <div className="card wide">
      <h2>What will you cultivate?</h2>
      <p className="subtitle">Add the habits you want to track.</p>
      
      <div className="habit-list">
        {habits.map((h, i) => (
          <div key={i} className="habit-item">
            <span><strong>{h.name}</strong> ({h.goalMinutes} {h.unit})</span>
            {h.name !== 'Lotus Sit' && <button onClick={() => removeHabit(i)} className="delete-btn">×</button>}
          </div>
        ))}
      </div>

      <div className="add-habit-form">
        <input 
          placeholder="Habit Name (e.g. Reading)" 
          value={newHabit.name} 
          onChange={e => setNewHabit({...newHabit, name: e.target.value})} 
        />
        <input 
          type="number" 
          placeholder="Goal" 
          value={newHabit.goalMinutes} 
          onChange={e => setNewHabit({...newHabit, goalMinutes: parseInt(e.target.value)})} 
          style={{width: '80px'}}
        />
        <input 
          placeholder="Unit (e.g. pages)" 
          value={newHabit.unit} 
          onChange={e => setNewHabit({...newHabit, unit: e.target.value})} 
          style={{width: '100px'}}
        />
        <button type="button" onClick={addHabit} className="secondary-btn">Add</button>
      </div>

      {error && <p className="error">{error}</p>}
      
      <div className="actions">
        <button onClick={onBack} className="text-btn">Back</button>
        <button onClick={handleRegister} className="primary-btn">Start Program</button>
      </div>
    </div>
  );
}

function Login({ onSuccess, onSwitchToRegister }) {
  const [data, setData] = useState({ username: '', password: '' });
  const [error, setError] = useState('');

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const res = await axios.post(`${API_BASE}/login`, data);
      onSuccess(res.data.token);
    } catch (err) {
      setError(err.response?.data?.error || 'Login failed');
    }
  };

  return (
    <div className="card">
      <h2>Welcome Back</h2>
      <form onSubmit={handleSubmit}>
        <input name="username" placeholder="Username" onChange={e => setData({...data, username: e.target.value})} />
        <input name="password" type="password" placeholder="Password" onChange={e => setData({...data, password: e.target.value})} />
        <button type="submit" className="primary-btn">Login</button>
      </form>
      {error && <p className="error">{error}</p>}
      <p className="switch-link" onClick={onSwitchToRegister}>Need an account? Register</p>
    </div>
  );
}

function Dashboard({ token }) {
  const [data, setData] = useState(null);

  const fetchData = async () => {
    try {
      const res = await axios.get(`${API_BASE}/daily-check-in`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      setData(res.data);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => { fetchData(); }, [token]);

  const completeTask = async (habitId, minutes) => {
    try {
      await axios.post(`${API_BASE}/complete-task`, 
        { habitId, minutes }, 
        { headers: { Authorization: `Bearer ${token}` } }
      );
      fetchData();
    } catch (err) {
      console.error(err);
    }
  };

  if (!data) return <div className="loading">Loading your Lotus...</div>;

  return (
    <div className="dashboard">
      <div className="status-card">
        <div className="phase-badge">{data.phase} Phase</div>
        <h2>Day {data.dayInProgram}</h2>
        <div className="lotus-visual">
          <div className={`lotus-icon ${data.lotusStatus}`}>🪷</div>
          <p>Status: {data.lotusStatus} ({data.growthPercent}%)</p>
        </div>
      </div>

      <h3>Today's Tasks</h3>
      <div className="task-list">
        {data.habits.map(h => (
          <div key={h.id} className={`task-card ${h.completed ? 'completed' : ''}`}>
            <div className="task-info">
              <h4>{h.name}</h4>
              <p>Target: {h.currentMinutes} {h.unit}</p>
            </div>
            {!h.completed ? (
              <button onClick={() => completeTask(h.id, h.currentMinutes)} className="check-btn">Check</button>
            ) : (
              <div className="done-badge">✓ Done</div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

export default App;