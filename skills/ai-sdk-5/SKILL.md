---
name: ai-sdk-5
description: >
  Vercel AI SDK 5 patterns.
  Trigger: When building AI chat features - breaking changes from v4.
metadata:
  author: mio
  version: "1.0"
---

## Breaking Changes from AI SDK 4

```typescript
// ❌ AI SDK 4 (OLD)
import { useChat } from "ai";
const { messages, handleSubmit } = useChat({ api: "/api/chat" });

// ✅ AI SDK 5 (NEW)
import { useChat } from "@ai-sdk/react";
import { DefaultChatTransport } from "ai";

const { messages, sendMessage } = useChat({
  transport: new DefaultChatTransport({ api: "/api/chat" }),
});
```

## Client Setup

```typescript
import { useChat } from "@ai-sdk/react";
import { DefaultChatTransport } from "ai";
import { useState } from "react";

export function Chat() {
  const [input, setInput] = useState("");
  const { messages, sendMessage, isLoading, error } = useChat({
    transport: new DefaultChatTransport({ api: "/api/chat" }),
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim()) return;
    sendMessage({ text: input });
    setInput("");
  };

  return (
    <form onSubmit={handleSubmit}>
      {messages.map((msg) => <Message key={msg.id} message={msg} />)}
      <input value={input} onChange={(e) => setInput(e.target.value)} />
      <button type="submit" disabled={isLoading}>Send</button>
    </form>
  );
}
```

## UIMessage Parts (v5)

```typescript
// message.parts is an array (not .content string)
type MessagePart =
  | { type: "text"; text: string }
  | { type: "image"; image: string }
  | { type: "tool-call"; toolCallId: string; toolName: string; args: unknown }
  | { type: "tool-result"; toolCallId: string; result: unknown };

function getMessageText(message: UIMessage): string {
  return message.parts
    .filter((p): p is { type: "text"; text: string } => p.type === "text")
    .map((p) => p.text)
    .join("");
}
```

## Server-Side (Route Handler)

```typescript
// app/api/chat/route.ts
import { openai } from "@ai-sdk/openai";
import { streamText } from "ai";

export async function POST(req: Request) {
  const { messages } = await req.json();
  const result = await streamText({
    model: openai("gpt-4o"),
    messages,
    system: "You are a helpful assistant.",
  });
  return result.toDataStreamResponse();
}
```

## Streaming with Tools

```typescript
import { streamText, tool } from "ai";
import { z } from "zod";

const result = await streamText({
  model: openai("gpt-4o"),
  messages,
  tools: {
    getWeather: tool({
      description: "Get weather for a location",
      parameters: z.object({
        location: z.string().describe("City name"),
      }),
      execute: async ({ location }) => {
        return { temperature: 72, condition: "sunny" };
      },
    }),
  },
});
```

## Keywords
ai sdk, vercel ai, chat, streaming, langchain, openai, llm
