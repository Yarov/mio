---
name: zod-4
description: >
  Zod 4 schema validation patterns.
  Trigger: When using Zod for validation - breaking changes from v3.
metadata:
  author: mio
  version: "1.0"
---

## Breaking Changes from Zod 3

```typescript
// ❌ Zod 3 (OLD)
z.string().email()
z.string().uuid()
z.string().url()

// ✅ Zod 4 (NEW)
z.email()
z.uuid()
z.url()
z.string().min(1) // replaces .nonempty()
```

## Object Schemas

```typescript
const userSchema = z.object({
  id: z.uuid(),
  email: z.email({ error: "Invalid email address" }),
  name: z.string().min(1, { error: "Name is required" }),
  age: z.number().int().positive().optional(),
  role: z.enum(["admin", "user", "guest"]),
});

type User = z.infer<typeof userSchema>;

// Parsing
const user = userSchema.parse(data);         // Throws on error
const result = userSchema.safeParse(data);   // Returns { success, data/error }

if (result.success) {
  console.log(result.data);
} else {
  console.log(result.error.issues);
}
```

## Discriminated Unions

```typescript
const resultSchema = z.discriminatedUnion("status", [
  z.object({ status: z.literal("success"), data: z.unknown() }),
  z.object({ status: z.literal("error"), error: z.string() }),
]);
```

## Transformations & Coercion

```typescript
const lowercaseEmail = z.email().transform(email => email.toLowerCase());
const numberFromString = z.coerce.number();  // "42" → 42
const dateFromString = z.coerce.date();      // "2024-01-01" → Date
```

## Refinements

```typescript
const passwordSchema = z.string()
  .min(8)
  .refine(val => /[A-Z]/.test(val), { message: "Must contain uppercase" })
  .refine(val => /[0-9]/.test(val), { message: "Must contain number" });

const formSchema = z.object({
  password: z.string(),
  confirmPassword: z.string(),
}).superRefine((data, ctx) => {
  if (data.password !== data.confirmPassword) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: "Passwords don't match",
      path: ["confirmPassword"],
    });
  }
});
```

## Error Handling (Zod 4)

```typescript
// Use 'error' param instead of 'message'
const schema = z.object({
  name: z.string({ error: "Name must be a string" }),
  email: z.email({ error: "Invalid email format" }),
  age: z.number().min(18, { error: "Must be 18 or older" }),
});
```

## React Hook Form Integration

```typescript
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";

const schema = z.object({
  email: z.email(),
  password: z.string().min(8),
});

type FormData = z.infer<typeof schema>;

function Form() {
  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <input {...register("email")} />
      {errors.email && <span>{errors.email.message}</span>}
    </form>
  );
}
```

## Keywords
zod, validation, schema, typescript, forms, parsing
