// ~/.claude-hooks/scripts/protect_secrets.ts
// Example: deny writes to secret files via TypeScript script rule

export const events = ["PreToolUse"] as const;

export function decide(e: { tool_name: string; tool_input?: { file_path?: string } }): { permissionDecision: string; permissionDecisionReason?: string } | null {
    const fp = e.tool_input?.file_path ?? "";
    if (/\.env$|secrets\/|\.pem$|\.key$/.test(fp)) {
        return {
            permissionDecision: "deny",
            permissionDecisionReason: "禁止修改密钥文件: " + fp,
        };
    }
    return null;
}
