import type { Flag } from "@/types/flags";

export const dummyFlags: Flag[] = [
  {
    id: "550e8400-e29b-41d4-a716-446655440001",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "new-checkout-flow",
    description: "Enable the new checkout flow for premium users",
    enabled: true,
    rules: [
      {
        id: "rule-1",
        attribute: "user.plan",
        operator: "equals",
        value: "premium",
      },
      {
        id: "rule-2",
        attribute: "user.email_verified",
        operator: "equals",
        value: true,
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-20T10:00:00Z",
    updated_at: "2025-12-27T14:30:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440002",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "dark-mode",
    description: "Allow users to toggle dark mode in settings",
    enabled: true,
    rules: [],
    rule_logic: "AND",
    created_at: "2025-12-15T08:20:00Z",
    updated_at: "2025-12-15T08:20:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440003",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "ai-suggestions",
    description: "Show AI-powered suggestions in the editor",
    enabled: false,
    rules: [
      {
        id: "rule-3",
        attribute: "user.role",
        operator: "equals",
        value: "admin",
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-22T16:45:00Z",
    updated_at: "2025-12-26T09:15:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440004",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "beta-features",
    description: "Enable access to beta features for early adopters",
    enabled: true,
    rules: [
      {
        id: "rule-4",
        attribute: "user.beta_tester",
        operator: "equals",
        value: true,
      },
      {
        id: "rule-5",
        attribute: "user.account_age_days",
        operator: "greater_than",
        value: 30,
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-10T12:00:00Z",
    updated_at: "2025-12-25T18:00:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440005",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "notifications-v2",
    description: "New notification system with real-time updates",
    enabled: false,
    rules: [
      {
        id: "rule-6",
        attribute: "user.region",
        operator: "equals",
        value: "us-east",
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-18T11:30:00Z",
    updated_at: "2025-12-27T10:20:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440006",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "advanced-analytics",
    description: "Detailed analytics dashboard with custom metrics",
    enabled: true,
    rules: [
      {
        id: "rule-7",
        attribute: "user.plan",
        operator: "equals",
        value: "enterprise",
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-05T09:00:00Z",
    updated_at: "2025-12-20T15:45:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440007",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "mobile-app-banner",
    description: "Show mobile app download banner on mobile web",
    enabled: true,
    rules: [
      {
        id: "rule-8",
        attribute: "device.type",
        operator: "equals",
        value: "mobile",
      },
      {
        id: "rule-9",
        attribute: "user.has_mobile_app",
        operator: "equals",
        value: false,
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-12T14:20:00Z",
    updated_at: "2025-12-24T16:00:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440008",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "collaborative-editing",
    description: "Real-time collaborative editing for documents",
    enabled: false,
    rules: [
      {
        id: "rule-10",
        attribute: "user.plan",
        operator: "equals",
        value: "team",
      },
      {
        id: "rule-11",
        attribute: "user.plan",
        operator: "equals",
        value: "enterprise",
      },
    ],
    rule_logic: "OR",
    created_at: "2025-12-08T10:15:00Z",
    updated_at: "2025-12-26T12:30:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440009",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "payment-methods-v2",
    description: "New payment methods including crypto and buy-now-pay-later",
    enabled: false,
    rules: [
      {
        id: "rule-12",
        attribute: "user.country",
        operator: "equals",
        value: "US",
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-14T13:00:00Z",
    updated_at: "2025-12-27T08:45:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440010",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "social-sharing",
    description: "Share content directly to social media platforms",
    enabled: true,
    rules: [],
    rule_logic: "AND",
    created_at: "2025-12-06T15:30:00Z",
    updated_at: "2025-12-06T15:30:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440011",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "api-rate-limiting-v2",
    description: "Improved API rate limiting with tiered limits",
    enabled: true,
    rules: [
      {
        id: "rule-13",
        attribute: "api.version",
        operator: "greater_than",
        value: 2,
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-01T07:00:00Z",
    updated_at: "2025-12-23T11:00:00Z",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440012",
    project_id: "660e8400-e29b-41d4-a716-446655440001",
    name: "advanced-search",
    description: "Advanced search with filters and natural language queries",
    enabled: false,
    rules: [
      {
        id: "rule-14",
        attribute: "user.email",
        operator: "contains",
        value: "@company.com",
      },
    ],
    rule_logic: "AND",
    created_at: "2025-12-19T09:45:00Z",
    updated_at: "2025-12-27T13:20:00Z",
  },
];

export function getFlagById(id: string): Flag | undefined {
  return dummyFlags.find((flag) => flag.id === id);
}

export function getAllFlags(): Flag[] {
  return dummyFlags;
}
