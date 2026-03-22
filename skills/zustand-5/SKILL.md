---
name: zustand-5
description: >
  Zustand 5 state management patterns.
  Trigger: When managing React state with Zustand.
metadata:
  author: mio
  version: "1.0"
---

## Basic Store

```typescript
import { create } from "zustand";

interface CounterStore {
  count: number;
  increment: () => void;
  decrement: () => void;
  reset: () => void;
}

const useCounterStore = create<CounterStore>((set) => ({
  count: 0,
  increment: () => set((state) => ({ count: state.count + 1 })),
  decrement: () => set((state) => ({ count: state.count - 1 })),
  reset: () => set({ count: 0 }),
}));
```

## Selectors (REQUIRED)

```typescript
// ✅ Select specific fields to prevent unnecessary re-renders
function UserName() {
  const name = useUserStore((state) => state.name);
  return <span>{name}</span>;
}

// ✅ For multiple fields, use useShallow
import { useShallow } from "zustand/react/shallow";

function UserInfo() {
  const { name, email } = useUserStore(
    useShallow((state) => ({ name: state.name, email: state.email }))
  );
  return <div>{name} - {email}</div>;
}

// ❌ AVOID: Selecting entire store
const store = useUserStore();  // Re-renders on ANY state change
```

## Persist Middleware

```typescript
import { persist } from "zustand/middleware";

const useSettingsStore = create<SettingsStore>()(
  persist(
    (set) => ({
      theme: "light",
      setTheme: (theme) => set({ theme }),
    }),
    { name: "settings-storage" }
  )
);
```

## Async Actions

```typescript
const useUserStore = create<UserStore>((set) => ({
  user: null,
  loading: false,
  error: null,
  fetchUser: async (id) => {
    set({ loading: true, error: null });
    try {
      const response = await fetch(`/api/users/${id}`);
      const user = await response.json();
      set({ user, loading: false });
    } catch (error) {
      set({ error: "Failed to fetch user", loading: false });
    }
  },
}));
```

## Slices Pattern

```typescript
const createUserSlice = (set): UserSlice => ({
  user: null,
  setUser: (user) => set({ user }),
});

const createCartSlice = (set): CartSlice => ({
  items: [],
  addItem: (item) => set((state) => ({ items: [...state.items, item] })),
});

type Store = UserSlice & CartSlice;

const useStore = create<Store>()((...args) => ({
  ...createUserSlice(...args),
  ...createCartSlice(...args),
}));
```

## Immer Middleware

```typescript
import { immer } from "zustand/middleware/immer";

const useTodoStore = create<TodoStore>()(
  immer((set) => ({
    todos: [],
    addTodo: (text) => set((state) => {
      state.todos.push({ id: crypto.randomUUID(), text, done: false });
    }),
    toggleTodo: (id) => set((state) => {
      const todo = state.todos.find(t => t.id === id);
      if (todo) todo.done = !todo.done;
    }),
  }))
);
```

## Outside React

```typescript
const { count, increment } = useCounterStore.getState();
increment();

const unsubscribe = useCounterStore.subscribe(
  (state) => console.log("Count changed:", state.count)
);
```

## Keywords
zustand, state management, react, store, persist, middleware
