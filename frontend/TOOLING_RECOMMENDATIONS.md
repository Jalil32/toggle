# Tooling Recommendations for Toggle Frontend

Based on the current stack analysis, here are tools that could improve the codebase:

## Testing Stack
Currently **missing** - this is the biggest gap:

### Unit & Integration Testing
- **Vitest** - Fast, modern test runner (better than Jest for Vite/Next.js)
- **React Testing Library** - Component testing
- **MSW (Mock Service Worker)** - API mocking for tests

### E2E Testing
- **Playwright** - End-to-end testing (recommended for Next.js)
  - Or **Cypress** - Alternative with better DX for some teams

### Example setup
```bash
npm install -D vitest @testing-library/react @testing-library/jest-dom
npm install -D @playwright/test
npm install -D msw
```

---

## Type Safety Enhancements

### Database
- **Drizzle ORM** - Type-safe SQL with migrations (lighter than Prisma)
  - Or **Prisma** - Full-featured ORM with great TypeScript support
  - You're using raw `pg` - an ORM would give you migrations + type safety

### API Type Safety
- **tRPC** - End-to-end type safety between frontend/backend
- **Zod** (you have it) - Extend usage for API validation
- **ts-rest** - Type-safe REST APIs (alternative to tRPC)

### Runtime Validation
Extend **Zod** usage beyond forms to environment variables:
```typescript
// lib/env.ts
import { z } from 'zod'

const envSchema = z.object({
  NEXT_PUBLIC_API_URL: z.string().url(),
  DATABASE_URL: z.string(),
  RESEND_API_KEY: z.string(),
})

export const env = envSchema.parse(process.env)
```

---

## Error Tracking & Monitoring

### Production Monitoring
- **Sentry** - Error tracking & performance monitoring
- **Vercel Analytics** - If deploying to Vercel (built-in)
- **Posthog** - Product analytics + feature flags

### Logging
- **Pino** - Fast JSON logger for Node.js
- **Better Stack** (formerly Logtail) - Log management

---

## Database & Backend

### ORM/Query Builder
```bash
# You're using pg directly - consider:
npm install drizzle-orm drizzle-kit
# Or
npm install prisma @prisma/client
```

### Migrations
- **Drizzle Kit** - SQL migrations
- **Prisma Migrate** - Schema migrations
- Currently you likely have no migration system

### Database Tools
- **Kysely** (already implied by Better Auth) - Type-safe query builder

---

## Developer Experience

### Documentation
- **Storybook** - Component documentation & isolation
- **TypeDoc** - API documentation from TypeScript

### Git Hooks
- **Husky** - Git hooks (pre-commit, pre-push)
- **lint-staged** - Run linters on staged files only
  ```json
  {
    "*.{js,ts,tsx}": ["biome check --write"],
    "*.{json,md}": ["biome format --write"]
  }
  ```

### Environment Management
- **T3 Env** - Type-safe environment variables
- **dotenv-cli** - Multiple env file support

---

## Performance & Optimization

### Bundle Analysis
- **@next/bundle-analyzer** - Visualize bundle size
  ```bash
  npm install -D @next/bundle-analyzer
  ```

### Image Optimization
- You have `next/image` - ensure you're using it
- **sharp** - Already auto-installed by Next.js

### Performance Monitoring
- **Next.js Speed Insights** (Vercel)
- **Lighthouse CI** - Automated performance checks

---

## API & Backend Communication

### API Client
- **TanStack Query** (React Query) - Server state management
  ```bash
  npm install @tanstack/react-query
  ```
  - Caching, refetching, optimistic updates
  - Better than manual `fetch` in components

### Type Safety
- **openapi-typescript** - Generate types from OpenAPI specs
- **tRPC** - If you control the backend

---

## CI/CD & Automation

### GitHub Actions
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm ci
      - run: npm run lint
      - run: npm run test
      - run: npm run build
```

### Pre-deployment Checks
- **Vercel Pre-deploy Hooks**
- **Chromatic** - Visual regression testing for Storybook

---

## Security

### Dependency Scanning
- **npm audit** - Built-in
- **Snyk** - Advanced vulnerability scanning
- **Dependabot** - Automated dependency updates (GitHub)

### Secret Scanning
- **git-secrets** - Prevent committing secrets
- **Gitleaks** - Scan for hardcoded secrets

### Security Headers
```typescript
// next.config.ts
export default {
  async headers() {
    return [{
      source: '/:path*',
      headers: [
        { key: 'X-Frame-Options', value: 'DENY' },
        { key: 'X-Content-Type-Options', value: 'nosniff' },
        // ... more security headers
      ],
    }]
  },
}
```

---

## Feature Flags (Related to Your Product)

Since you're building a feature flag platform:
- **OpenFeature** - Standard feature flag SDK
- **Flagsmith** - Open-source feature flags
- Study how competitors structure their SDKs

---

## Recommended Priority Order

### High Priority (add ASAP)
1. **Testing** (Vitest + Playwright)
2. **ORM** (Drizzle or Prisma) - Replace raw `pg`
3. **Error Tracking** (Sentry)
4. **Git Hooks** (Husky + lint-staged)

### Medium Priority
5. **TanStack Query** - Better API state management
6. **Bundle Analyzer** - Monitor bundle size
7. **Environment Validation** - Type-safe env vars

### Nice to Have
8. Storybook - Component docs
9. Lighthouse CI - Performance monitoring
10. OpenAPI types - If backend has spec

---

## Quick Wins

Add these to `package.json`:
```json
{
  "scripts": {
    "test": "vitest",
    "test:e2e": "playwright test",
    "analyze": "ANALYZE=true next build",
    "type-check": "tsc --noEmit",
    "prepare": "husky install"
  }
}
```

---

## Current Stack (For Reference)

**Core**: Next.js 16.1, React 19, TypeScript 5
**Auth**: Better Auth (recently migrated from Auth0)
**UI**: Tailwind CSS 4, Radix UI, Shadcn/ui
**Forms**: React Hook Form + Zod validation
**Charts**: Recharts
**Code Quality**: Biome (linter/formatter)
**Database**: PostgreSQL (raw `pg` driver)
