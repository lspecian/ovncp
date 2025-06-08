import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuthStore } from '@/stores/auth';
import { Github, Chrome, KeyRound } from 'lucide-react';

export default function LoginPage() {
  const navigate = useNavigate();
  const { user, login, isLoading, error } = useAuthStore();

  useEffect(() => {
    if (user) {
      navigate('/dashboard');
    }
  }, [user, navigate]);

  const handleLogin = (provider: string) => {
    login(provider);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-2xl font-bold text-center">
            OVN Control Platform
          </CardTitle>
          <CardDescription className="text-center">
            Sign in to manage your Open Virtual Network infrastructure
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {error && (
            <div className="bg-destructive/10 border border-destructive/20 text-destructive rounded-md p-3 text-sm">
              {error}
            </div>
          )}
          
          <div className="space-y-2">
            <Button
              variant="outline"
              className="w-full"
              onClick={() => handleLogin('github')}
              disabled={isLoading}
            >
              <Github className="mr-2 h-4 w-4" />
              Continue with GitHub
            </Button>
            
            <Button
              variant="outline"
              className="w-full"
              onClick={() => handleLogin('google')}
              disabled={isLoading}
            >
              <Chrome className="mr-2 h-4 w-4" />
              Continue with Google
            </Button>
            
            {/* Show custom OIDC option if configured */}
            {import.meta.env.VITE_OIDC_ENABLED === 'true' && (
              <Button
                variant="outline"
                className="w-full"
                onClick={() => handleLogin('oidc')}
                disabled={isLoading}
              >
                <KeyRound className="mr-2 h-4 w-4" />
                Continue with SSO
              </Button>
            )}
          </div>
          
          <div className="text-center text-sm text-muted-foreground">
            By signing in, you agree to our terms of service and privacy policy.
          </div>
        </CardContent>
      </Card>
    </div>
  );
}