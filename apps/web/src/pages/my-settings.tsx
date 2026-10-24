import { AccountActions } from "@/components/auth/account-actions";

// MySettingsPage is the personal settings surface: password change and sign out.
export default function MySettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <AccountActions />
    </div>
  );
}
