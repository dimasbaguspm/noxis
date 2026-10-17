import { AccountActions } from "@/components/auth/account-actions";

// MySettingsPage is the personal settings surface: password change and sign out.
export default function MySettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <h1 className="text-base font-medium">Settings</h1>
        <p className="text-xs text-muted-foreground">Your password and your session.</p>
      </div>
      <AccountActions />
    </div>
  );
}
