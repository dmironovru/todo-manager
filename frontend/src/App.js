import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';

const API_BASE = '/api/tasks';

function App() {
  const [tasks, setTasks] = useState([]);
  const [newTask, setNewTask] = useState('');
  const [editingId, setEditingId] = useState(null);
  const [editText, setEditText] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    fetchTasks();
  }, []);

  const fetchTasks = async () => {
    try {
      const res = await axios.get(API_BASE);
      // Защита: убеждаемся, что данные — массив
      if (Array.isArray(res.data)) {
        setTasks(res.data);
      } else {
        console.error('API вернул не массив:', res.data);
        setTasks([]);
      }
      setErrorMessage('');
    } catch (err) {
      console.error('Ошибка загрузки:', err);
      setErrorMessage('Ошибка загрузки задач');
      setTasks([]);
    }
  };

  const addTask = async () => {
    if (!newTask.trim()) return;
    try {
      await axios.post(API_BASE, { title: newTask });
      setNewTask('');
      fetchTasks();
    } catch (err) {
      console.error('Ошибка добавления:', err);
      setErrorMessage('Ошибка добавления задачи');
    }
  };

  const deleteTask = async (id) => {
    try {
      await axios.delete(`${API_BASE}/${id}`);
      fetchTasks();
    } catch (err) {
      console.error('Ошибка удаления:', err);
      setErrorMessage('Ошибка удаления задачи');
    }
  };

  const updateTask = async (id, completed) => {
    try {
      await axios.put(`${API_BASE}/${id}`, { completed });
      fetchTasks();
    } catch (err) {
      console.error('Ошибка обновления:', err);
      setErrorMessage('Ошибка обновления задачи');
    }
  };

  const startEdit = (task) => {
    setEditingId(task.id);
    setEditText(task.title);
  };

  const saveEdit = async (id) => {
    try {
      await axios.put(`${API_BASE}/${id}`, { title: editText });
      setEditingId(null);
      fetchTasks();
    } catch (err) {
      console.error('Ошибка сохранения:', err);
      setErrorMessage('Ошибка сохранения задачи');
    }
  };

  const clearAllTasks = async () => {
    if (window.confirm('Удалить все задачи? Это действие нельзя отменить.')) {
      try {
        await axios.delete(API_BASE);
        fetchTasks();
        setErrorMessage('');
      } catch (err) {
        console.error('Ошибка очистки:', err);
        setErrorMessage('Ошибка очистки задач');
      }
    }
  };

  // Защита: если tasks не массив, показываем 0
  const completedCount = Array.isArray(tasks) ? tasks.filter(t => t.completed).length : 0;
  const tasksLength = Array.isArray(tasks) ? tasks.length : 0;

  return (
    <div className="App">
      <div className="header">
        <h1>📋 Менеджер задач</h1>
        <div className="subtitle">React + Go + PostgreSQL + Docker</div>
      </div>

      <div className="add-form">
        <input
          type="text"
          value={newTask}
          onChange={(e) => setNewTask(e.target.value)}
          placeholder="Новая задача..."
          onKeyPress={(e) => e.key === 'Enter' && addTask()}
        />
        <button onClick={addTask}>➕ Добавить</button>
      </div>

      {errorMessage && (
        <div className="error-message">{errorMessage}</div>
      )}

      <div className="stats-bar">
        <span className="task-count">📌 Всего: {tasksLength} | ✅ Выполнено: {completedCount}</span>
        {tasksLength > 0 && (
          <button className="clear-btn" onClick={clearAllTasks}>🗑 Очистить всё</button>
        )}
      </div>

      <ul className="tasks-list">
        {Array.isArray(tasks) && tasks.map(task => (
          <li key={task.id} className={task.completed ? 'completed' : ''}>
            {editingId === task.id ? (
              <input
                className="edit-input"
                type="text"
                value={editText}
                onChange={(e) => setEditText(e.target.value)}
                onBlur={() => saveEdit(task.id)}
                onKeyPress={(e) => e.key === 'Enter' && saveEdit(task.id)}
                autoFocus
              />
            ) : (
              <>
                <input
                  className="task-checkbox"
                  type="checkbox"
                  checked={task.completed}
                  onChange={() => updateTask(task.id, !task.completed)}
                />
                <span className="task-title" onDoubleClick={() => startEdit(task)}>
                  {task.title}
                </span>
                <button className="edit-btn" onClick={() => startEdit(task)}>✏️</button>
                <button className="delete-btn" onClick={() => deleteTask(task.id)}>🗑️</button>
              </>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default App;