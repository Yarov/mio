---
name: playwright
description: >
  Playwright E2E testing patterns.
  Trigger: When writing E2E tests - Page Objects, selectors, test structure.
metadata:
  author: mio
  version: "1.0"
---

## File Structure

```
tests/
├── base-page.ts              # Parent class for ALL pages
├── helpers.ts                # Shared utilities
└── {page-name}/
    ├── {page-name}-page.ts   # Page Object Model
    └── {page-name}.spec.ts   # ALL tests for this page (one file!)
```

## Selector Priority (REQUIRED)

```typescript
// 1. BEST - getByRole
this.submitButton = page.getByRole("button", { name: "Submit" });

// 2. BEST - getByLabel
this.emailInput = page.getByLabel("Email");

// 3. SPARINGLY - getByText (static content only)
this.errorMessage = page.getByText("Invalid credentials");

// 4. LAST RESORT - getByTestId
this.customWidget = page.getByTestId("date-picker");

// ❌ AVOID fragile selectors
this.button = page.locator(".btn-primary");  // NO
this.input = page.locator("#email");         // NO
```

## Page Object Pattern

```typescript
import { Page, Locator, expect } from "@playwright/test";

export class BasePage {
  constructor(protected page: Page) {}

  async goto(path: string): Promise<void> {
    await this.page.goto(path);
    await this.page.waitForLoadState("networkidle");
  }

  async waitForNotification(): Promise<void> {
    await this.page.waitForSelector('[role="status"]');
  }
}

export class LoginPage extends BasePage {
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;

  constructor(page: Page) {
    super(page);
    this.emailInput = page.getByLabel("Email");
    this.passwordInput = page.getByLabel("Password");
    this.submitButton = page.getByRole("button", { name: "Sign in" });
  }

  async goto(): Promise<void> {
    await super.goto("/login");
  }

  async login(email: string, password: string): Promise<void> {
    await this.emailInput.fill(email);
    await this.passwordInput.fill(password);
    await this.submitButton.click();
  }
}
```

## Page Object Reuse (CRITICAL)

```typescript
// ✅ GOOD: Reuse existing page objects
import { SignInPage } from "../sign-in/sign-in-page";
import { HomePage } from "../home/home-page";

test("User can sign up and login", async ({ page }) => {
  const signUpPage = new SignUpPage(page);
  const signInPage = new SignInPage(page);  // REUSE
  const homePage = new HomePage(page);      // REUSE

  await signUpPage.signUp(userData);
  await homePage.verifyPageLoaded();
  await signInPage.login(credentials);
});

// ❌ BAD: Recreating existing functionality
export class SignUpPage extends BasePage {
  async logout() { /* ... */ }  // HomePage already has this!
}
```

## Test Pattern with Tags

```typescript
test.describe("Login", () => {
  test("User can login successfully",
    { tag: ["@critical", "@e2e", "@login"] },
    async ({ page }) => {
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login("user@test.com", "pass123");
      await expect(page).toHaveURL("/dashboard");
    }
  );
});
```

## Commands

```bash
npx playwright test                    # Run all
npx playwright test --grep "login"     # Filter by name
npx playwright test --ui               # Interactive UI
npx playwright test --debug            # Debug mode
```

## Keywords
playwright, e2e, testing, page object model, selectors, end-to-end
