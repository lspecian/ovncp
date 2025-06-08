import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { useToast } from '@/components/ui/use-toast';
import { api } from '@/lib/api';
import { LogicalSwitch } from '@/types';
import { Loader2 } from 'lucide-react';

const switchSchema = z.object({
  name: z.string().min(1, 'Name is required').regex(/^[a-zA-Z0-9_-]+$/, 'Name can only contain letters, numbers, hyphens, and underscores'),
  other_config: z.string().optional(),
  external_ids: z.string().optional(),
});

type SwitchFormData = z.infer<typeof switchSchema>;

interface SwitchDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  switchData?: LogicalSwitch | null;
  isEdit?: boolean;
}

export default function SwitchDialog({
  open,
  onOpenChange,
  switchData,
  isEdit = false,
}: SwitchDialogProps) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  
  const form = useForm<SwitchFormData>({
    resolver: zodResolver(switchSchema),
    defaultValues: {
      name: '',
      other_config: '',
      external_ids: '',
    },
  });

  useEffect(() => {
    if (switchData && isEdit) {
      form.reset({
        name: switchData.name,
        other_config: switchData.other_config ? JSON.stringify(switchData.other_config, null, 2) : '',
        external_ids: switchData.external_ids ? JSON.stringify(switchData.external_ids, null, 2) : '',
      });
    } else {
      form.reset({
        name: '',
        other_config: '',
        external_ids: '',
      });
    }
  }, [switchData, isEdit, form]);

  const mutation = useMutation({
    mutationFn: async (data: SwitchFormData) => {
      const payload = {
        name: data.name,
        other_config: data.other_config ? JSON.parse(data.other_config) : {},
        external_ids: data.external_ids ? JSON.parse(data.external_ids) : {},
      };
      
      if (isEdit && switchData) {
        return api.updateLogicalSwitch(switchData.uuid, payload);
      } else {
        return api.createLogicalSwitch(payload);
      }
    },
    onSuccess: () => {
      toast({
        title: 'Success',
        description: `Logical switch ${isEdit ? 'updated' : 'created'} successfully`,
      });
      queryClient.invalidateQueries({ queryKey: ['switches'] });
      onOpenChange(false);
    },
    onError: (error) => {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to save switch',
        variant: 'destructive',
      });
    },
  });

  const onSubmit = (data: SwitchFormData) => {
    try {
      // Validate JSON fields
      if (data.other_config) {
        JSON.parse(data.other_config);
      }
      if (data.external_ids) {
        JSON.parse(data.external_ids);
      }
      mutation.mutate(data);
    } catch (error) {
      toast({
        title: 'Invalid JSON',
        description: 'Please ensure Other Config and External IDs are valid JSON',
        variant: 'destructive',
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[525px]">
        <DialogHeader>
          <DialogTitle>
            {isEdit ? 'Edit Logical Switch' : 'Create Logical Switch'}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update the logical switch configuration.'
              : 'Create a new logical switch in your OVN network.'}
          </DialogDescription>
        </DialogHeader>
        
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="my-switch" {...field} />
                  </FormControl>
                  <FormDescription>
                    A unique name for the logical switch
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <FormField
              control={form.control}
              name="other_config"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Other Config (JSON)</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder='{&#10;  "key": "value"&#10;}'
                      className="font-mono text-sm"
                      rows={4}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    Additional configuration as JSON (optional)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <FormField
              control={form.control}
              name="external_ids"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>External IDs (JSON)</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder='{&#10;  "key": "value"&#10;}'
                      className="font-mono text-sm"
                      rows={4}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    External identifiers as JSON (optional)
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                {isEdit ? 'Update' : 'Create'}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
}