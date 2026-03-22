---
name: tailwind-4
description: >
  Tailwind CSS 4 patterns and best practices.
  Trigger: When styling with Tailwind - cn(), theme variables, no var() in className.
metadata:
  author: mio
  version: "1.0"
---

## Styling Decision Tree

```
Tailwind class exists?  → className="..."
Dynamic value?          → style={{ width: `${x}%` }}
Conditional styles?     → cn("base", condition && "variant")
Static only?            → className="..." (no cn() needed)
Library can't use class?→ style prop with var() constants
```

## Critical Rules

### Never Use var() in className

```typescript
// ❌ NEVER
<div className="bg-[var(--color-primary)]" />

// ✅ ALWAYS: Tailwind semantic classes
<div className="bg-primary" />
```

### Never Use Hex Colors

```typescript
// ❌ NEVER
<p className="text-[#ffffff]" />

// ✅ ALWAYS: Tailwind color classes
<p className="text-white" />
```

## The cn() Utility

```typescript
import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

### When to Use cn()

```typescript
// ✅ Conditional classes
<div className={cn("base-class", isActive && "active-class")} />

// ✅ Merging with potential conflicts
<button className={cn("px-4 py-2", className)} />

// ✅ Multiple conditions
<div className={cn(
  "rounded-lg border",
  variant === "primary" && "bg-blue-500 text-white",
  variant === "secondary" && "bg-gray-200 text-gray-800",
  disabled && "opacity-50 cursor-not-allowed"
)} />

// ❌ Static classes - unnecessary wrapper
<div className={cn("flex items-center gap-2")} />
// ✅ Just use className directly
<div className="flex items-center gap-2" />
```

## Common Patterns

```typescript
// Flexbox
<div className="flex items-center justify-between gap-4" />

// Grid
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6" />

// Responsive
<div className="w-full md:w-1/2 lg:w-1/3" />
<div className="hidden md:block" />

// States
<button className="hover:bg-blue-600 focus:ring-2 active:scale-95" />

// Dark Mode
<div className="bg-white dark:bg-slate-900" />
```

## Keywords
tailwind, css, styling, cn, utility classes, responsive
