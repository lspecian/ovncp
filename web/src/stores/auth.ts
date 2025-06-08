import { create } from 'zustand';
import { Session, User } from '@/types';
import { api } from '@/lib/api';

interface AuthState {
  session: Session | null;
  user: User | null;
  isLoading: boolean;
  error: string | null;
  
  // Actions
  login: (provider: string) => Promise<void>;
  handleCallback: (provider: string, code: string, state: string) => Promise<void>;
  logout: () => Promise<void>;
  checkSession: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>((set, get) => ({
  session: null,
  user: null,
  isLoading: false,
  error: null,

  login: async (provider: string) => {
    try {
      set({ isLoading: true, error: null });
      const { auth_url } = await api.login(provider);
      // Redirect to OAuth provider
      window.location.href = auth_url;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Login failed' });
    } finally {
      set({ isLoading: false });
    }
  },

  handleCallback: async (provider: string, code: string, state: string) => {
    try {
      set({ isLoading: true, error: null });
      const session = await api.handleCallback(provider, code, state);
      set({ session, user: session.user });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : 'Authentication failed' });
      throw error;
    } finally {
      set({ isLoading: false });
    }
  },

  logout: async () => {
    try {
      set({ isLoading: true, error: null });
      await api.logout();
      set({ session: null, user: null });
    } catch (error) {
      // Even if logout fails on server, clear local state
      set({ session: null, user: null });
    } finally {
      set({ isLoading: false });
    }
  },

  checkSession: async () => {
    const stored = localStorage.getItem('ovncp_session');
    if (!stored) {
      set({ session: null, user: null });
      return;
    }

    try {
      const session = JSON.parse(stored) as Session;
      
      // Check if expired
      if (new Date().getTime() / 1000 > session.expires_at) {
        localStorage.removeItem('ovncp_session');
        set({ session: null, user: null });
        return;
      }

      // Verify with server
      const user = await api.getProfile();
      set({ session, user });
    } catch (error) {
      // Invalid session
      localStorage.removeItem('ovncp_session');
      api.setToken(null);
      set({ session: null, user: null });
    }
  },

  clearError: () => set({ error: null }),
}));