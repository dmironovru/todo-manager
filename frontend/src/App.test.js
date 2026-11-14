import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import axios from 'axios';
import App from './App';

jest.mock('axios');

const tasks = [
  { id: 1, title: 'Купить молоко', completed: false },
  { id: 2, title: 'Позвонить в банк', completed: true },
];

beforeEach(() => {
  jest.clearAllMocks();
  axios.get.mockResolvedValue({ data: tasks });
  axios.post.mockResolvedValue({ data: { id: 3, title: 'Новая', completed: false } });
  axios.put.mockResolvedValue({ data: {} });
  axios.delete.mockResolvedValue({ data: {} });
});

test('показывает задачи, загруженные с бэкенда', async () => {
  render(<App />);
  expect(await screen.findByText('Купить молоко')).toBeInTheDocument();
  expect(screen.getByText('Позвонить в банк')).toBeInTheDocument();
  expect(screen.getByText(/Всего: 2 \| ✅ Выполнено: 1/)).toBeInTheDocument();
});

test('добавление задачи отправляет POST с названием', async () => {
  render(<App />);
  await screen.findByText('Купить молоко');

  fireEvent.change(screen.getByPlaceholderText('Новая задача...'), {
    target: { value: 'Вынести мусор' },
  });
  fireEvent.click(screen.getByText('➕ Добавить'));

  await waitFor(() =>
    expect(axios.post).toHaveBeenCalledWith('/api/tasks', { title: 'Вынести мусор' })
  );
});

test('пустое название не отправляется', async () => {
  render(<App />);
  await screen.findByText('Купить молоко');

  fireEvent.change(screen.getByPlaceholderText('Новая задача...'), {
    target: { value: '   ' },
  });
  fireEvent.click(screen.getByText('➕ Добавить'));

  expect(axios.post).not.toHaveBeenCalled();
});

// Регрессия к багу «галочка слетает при переименовании»:
// фронт шлёт только title, поэтому бэкенд обязан не трогать completed.
test('переименование отправляет только title, без completed', async () => {
  render(<App />);
  const title = await screen.findByText('Купить молоко');

  fireEvent.doubleClick(title);
  const input = screen.getByDisplayValue('Купить молоко');
  fireEvent.change(input, { target: { value: 'Купить молоко и хлеб' } });
  fireEvent.blur(input);

  await waitFor(() =>
    expect(axios.put).toHaveBeenCalledWith('/api/tasks/1', { title: 'Купить молоко и хлеб' })
  );
  const [, payload] = axios.put.mock.calls[0];
  expect(payload).not.toHaveProperty('completed');
});

test('клик по чекбоксу отправляет только completed', async () => {
  render(<App />);
  await screen.findByText('Купить молоко');

  fireEvent.click(screen.getAllByRole('checkbox')[0]);

  await waitFor(() =>
    expect(axios.put).toHaveBeenCalledWith('/api/tasks/1', { completed: true })
  );
});

test('удаление задачи вызывает DELETE по её id', async () => {
  render(<App />);
  await screen.findByText('Купить молоко');

  fireEvent.click(screen.getAllByText('🗑️')[0]);

  await waitFor(() => expect(axios.delete).toHaveBeenCalledWith('/api/tasks/1'));
});

test('при ошибке загрузки показывает сообщение и не падает', async () => {
  axios.get.mockRejectedValueOnce(new Error('network'));
  const spy = jest.spyOn(console, 'error').mockImplementation(() => {});

  render(<App />);

  expect(await screen.findByText('Ошибка загрузки задач')).toBeInTheDocument();
  spy.mockRestore();
});
