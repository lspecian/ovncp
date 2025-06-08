import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from '@/components/ui/toaster';
import { useAuthStore } from '@/stores/auth';
import { ThemeProvider } from '@/components/theme-provider';

// Pages
import LoginPage from '@/pages/login';
import CallbackPage from '@/pages/callback';
import DashboardPage from '@/pages/dashboard';
import SwitchesPage from '@/pages/switches';
import RoutersPage from '@/pages/routers';
import PortsPage from '@/pages/ports';
import ACLsPage from '@/pages/acls';
import LoadBalancersPage from '@/pages/load-balancers';
import TopologyPage from '@/pages/topology';
import UsersPage from '@/pages/users';

// Components
import Layout from '@/components/layout';
import ProtectedRoute from '@/components/protected-route';

function App() {
  const checkSession = useAuthStore((state) => state.checkSession);

  useEffect(() => {
    checkSession();
  }, [checkSession]);

  return (
    <ThemeProvider defaultTheme="system" storageKey="ovncp-theme">
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/auth/callback/:provider" element={<CallbackPage />} />
          
          <Route element={<ProtectedRoute />}>
            <Route element={<Layout />}>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<DashboardPage />} />
              <Route path="/topology" element={<TopologyPage />} />
              <Route path="/switches" element={<SwitchesPage />} />
              <Route path="/routers" element={<RoutersPage />} />
              <Route path="/ports" element={<PortsPage />} />
              <Route path="/acls" element={<ACLsPage />} />
              <Route path="/load-balancers" element={<LoadBalancersPage />} />
              <Route path="/users" element={<UsersPage />} />
            </Route>
          </Route>
        </Routes>
        <Toaster />
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;