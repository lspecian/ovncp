import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useAuthStore } from '@/stores/auth';
import { Github, Chrome, KeyRound, User } from 'lucide-react';

export default function LoginPage() {
  const navigate = useNavigate();
  const { user, login, localLogin, isLoading, error } = useAuthStore();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  useEffect(() => {
    if (user) {
      navigate('/dashboard');
    }
  }, [user, navigate]);

  const handleLogin = (provider: string) => {
    login(provider);
  };

  const handleLocalLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await localLogin(username, password);
      navigate('/dashboard');
    } catch (error) {
      // Error is handled in the store
    }
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
          
          <Tabs defaultValue="local" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="local">Local Login</TabsTrigger>
              <TabsTrigger value="oauth">OAuth</TabsTrigger>
            </TabsList>
            
            <TabsContent value="local" className="space-y-4">
              <form onSubmit={handleLocalLogin} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="username">Username</Label>
                  <Input
                    id="username"
                    type="text"
                    placeholder="Enter your username"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    disabled={isLoading}
                    required
                  />
                </div>
                
                <div className="space-y-2">
                  <Label htmlFor="password">Password</Label>
                  <Input
                    id="password"
                    type="password"
                    placeholder="Enter your password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    disabled={isLoading}
                    required
                  />
                </div>
                
                <Button
                  type="submit"
                  className="w-full"
                  disabled={isLoading}
                >
                  <User className="mr-2 h-4 w-4" />
                  Sign In
                </Button>
                
                <div className="text-center text-sm text-muted-foreground">
                  Default credentials: admin / admin
                </div>
              </form>
            </TabsContent>
            
            <TabsContent value="oauth" className="space-y-4">
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
            </TabsContent>
          </Tabs>
          
          <div className="text-center text-sm text-muted-foreground">
            By signing in, you agree to our terms of service and privacy policy.
          </div>
        </CardContent>
      </Card>
    </div>
  );
}