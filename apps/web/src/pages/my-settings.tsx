import { AccountActions } from "@/components/auth/account-actions";
import { H1, P } from "@/components/ui/typography";

// MySettingsPage is the personal settings surface: password change and sign out.
export default function MySettingsPage() {
  return (
    <div className="flex flex-col gap-4">
      <div>
        <H1>Settings</H1>
        <P>Your password and your session.</P>
      </div>
      <AccountActions />
    </div>
  );
}
