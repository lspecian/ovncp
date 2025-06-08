import { useEffect } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { useAuthStore } from '@/stores/auth';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function CallbackPage() {
  const navigate = useNavigate();
  const { provider } = useParams();
  const [searchParams] = useSearchParams();
  const handleCallback = useAuthStore((state) => state.handleCallback);
  const error = useAuthStore((state) => state.error);
  
  useEffect(() => {
    const code = searchParams.get('code');
    const state = searchParams.get('state');
    const error = searchParams.get('error');
    
    if (error) {
      // OAuth provider returned an error
      navigate('/login');
      return;
    }
    
    if (!provider || !code || !state) {
      navigate('/login');
      return;
    }
    
    handleCallback(provider, code, state)
      .then(() => {
        navigate('/dashboard');
      })
      .catch(() => {
        // Error is handled in the store
        setTimeout(() => navigate('/login'), 3000);
      });
  }, [provider, searchParams, handleCallback, navigate]);
  
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-center">
            {error ? 'Authentication Failed' : 'Authenticating...'}
          </CardTitle>
        </CardHeader>
        <CardContent className="text-center">
          {error ? (
            <div className="text-destructive">{error}</div>
          ) : (
            <div className="text-muted-foreground">
              Please wait while we complete your sign in.
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}