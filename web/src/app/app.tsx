import { ConfigProvider, App as AntApp } from 'antd';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { Skeleton } from 'antd';
import { theme } from '../theme';
import { SessionProvider, useSession } from '../api/session-context';
import { PermissionProvider } from '../api/permissions';
import { AppShell } from './app-shell';
import { SignIn } from './sign-in';
import { TargetsPage } from '../pages/targets-page';
import { ErrorBoundary } from '../components/error-boundary';
// The instance and proposal screens bring Monaco with them; the list is what an operator opens every
// time, so it never pays for the editor.
const TargetPage = lazy(() =>
  import('../pages/target-page').then((m) => ({ default: m.TargetPage })),
);
const ProposalPage = lazy(() =>
  import('../pages/proposal-page').then((m) => ({ default: m.ProposalPage })),
);

export function App() {
  return (
    <ConfigProvider theme={theme}>
      <AntApp>
        <SessionProvider>
          <PermissionProvider>
            <BrowserRouter>
              <ErrorBoundary>
                <Gate />
              </ErrorBoundary>
            </BrowserRouter>
          </PermissionProvider>
        </SessionProvider>
      </AntApp>
    </ConfigProvider>
  );
}

function Gate() {
  const { boot } = useSession();

  // The sign-in screen is the whole window: a split needs the edges, and a topbar above a two-column
  // page would make the split look like a panel rather than a page.
  if (boot === 'anonymous') return <SignIn />;

  if (boot !== 'ready') {
    return (
      <AppShell>
        <Skeleton active paragraph={{ rows: 6 }} />
      </AppShell>
    );
  }

  return (
    <AppShell>
      <Suspense fallback={<Skeleton active paragraph={{ rows: 6 }} />}>
        <Routes>
          <Route path="/" element={<TargetsPage />} />
          <Route path="/targets/:ref" element={<TargetPage />} />
          <Route path="/proposals/:ref" element={<ProposalPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </AppShell>
  );
}
