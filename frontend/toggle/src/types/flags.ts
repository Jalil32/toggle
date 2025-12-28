export interface FlagRule {
  id: string;
  attribute: string;
  operator: "equals" | "not_equals" | "contains" | "greater_than" | "less_than";
  value: string | number | boolean;
}

export interface Flag {
  id: string;
  tenant_id: string;
  project_id?: string | null;
  name: string;
  description: string | null;
  enabled: boolean;
  rules: FlagRule[];
  rule_logic: "AND" | "OR";
  created_at: string;
  updated_at: string;
}
