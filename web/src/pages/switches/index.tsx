import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ColumnDef } from '@tanstack/react-table';
import { DataTable } from '@/components/ui/data-table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useToast } from '@/components/ui/use-toast';
import { api } from '@/lib/api';
import { LogicalSwitch } from '@/types';
import { useAuthStore } from '@/stores/auth';
import { Plus, Edit, Trash2, MoreHorizontal, Loader2 } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import SwitchDialog from './switch-dialog';

export default function SwitchesPage() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const user = useAuthStore((state) => state.user);
  const canEdit = user?.role === 'admin' || user?.role === 'operator';
  
  const [selectedSwitch, setSelectedSwitch] = useState<LogicalSwitch | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [isEdit, setIsEdit] = useState(false);

  // Fetch switches
  const { data, isLoading, error } = useQuery({
    queryKey: ['switches'],
    queryFn: () => api.listLogicalSwitches(100, 0),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteLogicalSwitch(id),
    onSuccess: () => {
      toast({
        title: 'Success',
        description: 'Logical switch deleted successfully',
      });
      queryClient.invalidateQueries({ queryKey: ['switches'] });
    },
    onError: (error) => {
      toast({
        title: 'Error',
        description: error instanceof Error ? error.message : 'Failed to delete switch',
        variant: 'destructive',
      });
    },
  });

  const handleCreate = () => {
    setSelectedSwitch(null);
    setIsEdit(false);
    setDialogOpen(true);
  };

  const handleEdit = (sw: LogicalSwitch) => {
    setSelectedSwitch(sw);
    setIsEdit(true);
    setDialogOpen(true);
  };

  const handleDelete = async (sw: LogicalSwitch) => {
    if (!confirm(`Are you sure you want to delete "${sw.name}"?`)) {
      return;
    }
    deleteMutation.mutate(sw.uuid);
  };

  const columns: ColumnDef<LogicalSwitch>[] = [
    {
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }) => (
        <div className="font-medium">{row.getValue('name')}</div>
      ),
    },
    {
      accessorKey: 'uuid',
      header: 'UUID',
      cell: ({ row }) => (
        <div className="font-mono text-xs text-muted-foreground">
          {row.getValue('uuid')}
        </div>
      ),
    },
    {
      id: 'ports',
      header: 'Ports',
      cell: ({ row }) => {
        const ports = row.original.ports || [];
        return <Badge variant="secondary">{ports.length} ports</Badge>;
      },
    },
    {
      id: 'acls',
      header: 'ACLs',
      cell: ({ row }) => {
        const acls = row.original.acls || [];
        return <Badge variant="secondary">{acls.length} rules</Badge>;
      },
    },
    {
      id: 'load_balancers',
      header: 'Load Balancers',
      cell: ({ row }) => {
        const lbs = row.original.load_balancers || [];
        return <Badge variant="secondary">{lbs.length} LBs</Badge>;
      },
    },
    {
      accessorKey: 'created_at',
      header: 'Created',
      cell: ({ row }) => {
        const date = row.getValue('created_at') as string;
        return date ? new Date(date).toLocaleDateString() : '-';
      },
    },
    {
      id: 'actions',
      cell: ({ row }) => {
        const sw = row.original;

        return (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => navigator.clipboard.writeText(sw.uuid)}
              >
                Copy UUID
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {canEdit && (
                <>
                  <DropdownMenuItem onClick={() => handleEdit(sw)}>
                    <Edit className="mr-2 h-4 w-4" />
                    Edit
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => handleDelete(sw)}
                    className="text-destructive"
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        );
      },
    },
  ];

  if (error) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-3xl font-bold">Logical Switches</h1>
          <p className="text-muted-foreground">Manage your logical switches</p>
        </div>
        <Card>
          <CardContent className="flex items-center justify-center h-32">
            <p className="text-destructive">Failed to load logical switches</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Logical Switches</h1>
          <p className="text-muted-foreground">Manage your logical switches</p>
        </div>
        {canEdit && (
          <Button onClick={handleCreate}>
            <Plus className="mr-2 h-4 w-4" />
            Create Switch
          </Button>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>All Switches</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center h-32">
              <Loader2 className="h-6 w-6 animate-spin" />
            </div>
          ) : (
            <DataTable
              columns={columns}
              data={data?.data || []}
              searchKey="name"
            />
          )}
        </CardContent>
      </Card>

      <SwitchDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        switchData={selectedSwitch}
        isEdit={isEdit}
      />
    </div>
  );
}