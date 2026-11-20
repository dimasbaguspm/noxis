import { ConfigProvider, App as AntApp } from 'antd';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { SessionProvider, useSession } from '../api/session';
import { theme } from '../theme';
import { AppShell } from './AppShell';
import { SignIn } from './SignIn';
import { lazy, Suspense } from 'react';
import { Skeleton } from 'antd';
import { TargetsPage } from '../pages/TargetsPage';

// The instance and proposal screens bring Monaco with them (~1 MB gzipped). The list screen is the
// one an operator opens every time, so it never pays for the editor.
const TargetPage = lazy(() => import('../pages/TargetPage').then((m) => ({ default: m.TargetPage })));
const ProposalPage = lazy(() => import('../pages/ProposalPage').then((m) => ({ default: m.ProposalPage })));

// The console has one shell and three routes. A screen that needs a session is simply not reachable
// without one, and the sign-in card is the only thing an anonymous visitor can see.

export function App() {
  return (
    <ConfigProvider theme={theme}>
      <AntApp>
        <SessionProvider>
          <BrowserRouter>
            <Gate />
          </BrowserRouter>
        </SessionProvider>
      </AntApp>
    </ConfigProvider>
  );
}

function Gate() {
  const { boot } = useSession();
  if (boot === 'loading') return <AppShell loading />;
  if (boot === 'anonymous' || boot === 'failed') return <SignIn />;
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
