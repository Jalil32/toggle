"use client";

import * as React from "react";
import {
    IconChevronLeft,
    IconChevronRight,
    IconChevronsLeft,
    IconChevronsRight,
} from "@tabler/icons-react";
import {
    flexRender,
    getCoreRowModel,
    getFacetedRowModel,
    getFacetedUniqueValues,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    useReactTable,
    type ColumnDef,
    type ColumnFiltersState,
    type SortingState,
    type VisibilityState,
} from "@tanstack/react-table";
import { useRouter } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import type { Flag } from "@/types/flags";

interface FlagsTableProps {
    data: Flag[];
    slug: string;
}

export function FlagsTable({ data, slug }: FlagsTableProps) {
    const router = useRouter();
    const [sorting, setSorting] = React.useState<SortingState>([]);
    const [columnFilters, setColumnFilters] =
        React.useState<ColumnFiltersState>([]);
    const [columnVisibility, setColumnVisibility] =
        React.useState<VisibilityState>({});
    const [rowSelection, setRowSelection] = React.useState({});
    const [statusFilter, setStatusFilter] = React.useState<string>("all");
    const [searchQuery, setSearchQuery] = React.useState<string>("");

    const columns: ColumnDef<Flag>[] = [
        {
            id: "content",
            header: () => null,
            cell: ({ row }) => {
                const description = row.original.description;
                return (
                    <div className="min-w-0 flex-1 flex items-center gap-4">
                        <div className="font-medium w-48 shrink-0">
                            {row.original.name}
                        </div>
                        {description && (
                            <div className="text-muted-foreground min-w-0 flex-1 truncate text-sm hidden md:block">
                                {description}
                            </div>
                        )}
                    </div>
                );
            },
            enableHiding: false,
        },
        {
            id: "metadata",
            header: () => null,
            cell: ({ row }) => {
                const enabled = row.original.enabled;
                const rules = row.original.rules;
                const date = new Date(row.original.updated_at);
                const now = new Date();
                const diffInMs = now.getTime() - date.getTime();
                const diffInDays = Math.floor(diffInMs / (1000 * 60 * 60 * 24));

                let timeAgo: string;
                if (diffInDays === 0) {
                    const diffInHours = Math.floor(diffInMs / (1000 * 60 * 60));
                    if (diffInHours === 0) {
                        const diffInMinutes = Math.floor(
                            diffInMs / (1000 * 60),
                        );
                        timeAgo =
                            diffInMinutes === 0
                                ? "Just now"
                                : `${diffInMinutes}m ago`;
                    } else {
                        timeAgo = `${diffInHours}h ago`;
                    }
                } else if (diffInDays === 1) {
                    timeAgo = "Yesterday";
                } else if (diffInDays < 7) {
                    timeAgo = `${diffInDays}d ago`;
                } else {
                    timeAgo = date.toLocaleDateString();
                }

                return (
                    <div className="flex items-center justify-end gap-2 flex-wrap select-none">
                        <Badge
                            variant="outline"
                            className="hidden sm:inline-flex"
                        >
                            {rules.length}{" "}
                            {rules.length === 1 ? "rule" : "rules"}
                        </Badge>
                        <Badge
                            variant="outline"
                            className="hidden md:inline-flex"
                        >
                            {timeAgo}
                        </Badge>
                        <StatusBadge
                            enabled={enabled}
                            className="touch-manipulation"
                        />
                    </div>
                );
            },
            enableHiding: false,
        },
    ];

    const filteredData = React.useMemo(() => {
        let filtered = data;

        // Filter by search query
        if (searchQuery) {
            filtered = filtered.filter(
                (flag) =>
                    flag.name
                        .toLowerCase()
                        .includes(searchQuery.toLowerCase()) ||
                    (flag.description &&
                        flag.description
                            .toLowerCase()
                            .includes(searchQuery.toLowerCase())),
            );
        }

        // Filter by status
        if (statusFilter === "enabled") {
            filtered = filtered.filter((flag) => flag.enabled);
        } else if (statusFilter === "disabled") {
            filtered = filtered.filter((flag) => !flag.enabled);
        }

        return filtered;
    }, [data, statusFilter, searchQuery]);

    const table = useReactTable({
        data: filteredData,
        columns,
        state: {
            sorting,
            columnVisibility,
            rowSelection,
            columnFilters,
        },
        enableRowSelection: true,
        onRowSelectionChange: setRowSelection,
        onSortingChange: setSorting,
        onColumnFiltersChange: setColumnFilters,
        onColumnVisibilityChange: setColumnVisibility,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: getFilteredRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        getSortedRowModel: getSortedRowModel(),
        getFacetedRowModel: getFacetedRowModel(),
        getFacetedUniqueValues: getFacetedUniqueValues(),
    });

    const handleRowClick = (flagId: string) => {
        router.push(`/${slug}/flags/${flagId}`);
    };

    return (
        <div className="flex h-full w-full flex-col gap-6">
            <div className="flex shrink-0 items-center justify-between gap-4 px-4 lg:px-6 select-none">
                <Input
                    placeholder="Search flags..."
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    className="max-w-sm transition-all duration-200 focus:shadow-lg focus:shadow-gradient-start/10"
                />
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                    <SelectTrigger className="w-[180px]" size="sm">
                        <SelectValue placeholder="Filter by status" />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">All Flags</SelectItem>
                        <SelectItem value="enabled">Enabled Only</SelectItem>
                        <SelectItem value="disabled">Disabled Only</SelectItem>
                    </SelectContent>
                </Select>
            </div>

            <div className="min-h-0 w-full flex-1 overflow-auto border-t">
                <Table>
                    <TableBody>
                        {table.getRowModel().rows?.length ? (
                            table.getRowModel().rows.map((row, index) => (
                                <TableRow
                                    key={row.id}
                                    data-state={
                                        row.getIsSelected() && "selected"
                                    }
                                    onClick={() =>
                                        handleRowClick(row.original.id)
                                    }
                                    className="cursor-pointer border-b transition-all duration-100 hover:bg-muted/50 group min-h-[64px] md:min-h-[auto] active:bg-muted/70 hover:shadow-[inset_4px_0_0_0_var(--accent)]"
                                >
                                    {row.getVisibleCells().map((cell) => (
                                        <TableCell
                                            key={cell.id}
                                            className="px-4 lg:px-6 py-4"
                                        >
                                            {flexRender(
                                                cell.column.columnDef.cell,
                                                cell.getContext(),
                                            )}
                                        </TableCell>
                                    ))}
                                </TableRow>
                            ))
                        ) : (
                            <TableRow className="hover:bg-transparent">
                                <TableCell
                                    colSpan={columns.length}
                                    className="h-24 px-4 text-center lg:px-6"
                                >
                                    <div className="flex flex-col items-center justify-center gap-2 text-muted-foreground">
                                        <p className="font-medium">
                                            No flags found.
                                        </p>
                                        <p className="text-sm">
                                            Create your first feature flag to
                                            get started.
                                        </p>
                                    </div>
                                </TableCell>
                            </TableRow>
                        )}
                    </TableBody>
                </Table>
            </div>

            {table.getPageCount() > 1 && (
                <div className="flex shrink-0 flex-col items-center justify-between gap-4 border-t px-4 pt-4 lg:flex-row lg:px-6 select-none">
                    <div className="text-muted-foreground text-sm">
                        {table.getFilteredSelectedRowModel().rows.length} of{" "}
                        {table.getFilteredRowModel().rows.length} row(s)
                        selected.
                    </div>
                    <div className="flex flex-col items-center gap-4 sm:flex-row">
                        <div className="flex items-center gap-2">
                            <Label
                                htmlFor="rows-per-page"
                                className="text-sm font-medium"
                            >
                                Rows per page
                            </Label>
                            <Select
                                value={`${table.getState().pagination.pageSize}`}
                                onValueChange={(value) => {
                                    table.setPageSize(Number(value));
                                }}
                            >
                                <SelectTrigger
                                    size="sm"
                                    className="w-20"
                                    id="rows-per-page"
                                >
                                    <SelectValue
                                        placeholder={
                                            table.getState().pagination.pageSize
                                        }
                                    />
                                </SelectTrigger>
                                <SelectContent side="top">
                                    {[10, 20, 30, 40, 50].map((pageSize) => (
                                        <SelectItem
                                            key={pageSize}
                                            value={`${pageSize}`}
                                        >
                                            {pageSize}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                        </div>
                        <div className="flex w-fit items-center justify-center text-sm font-medium">
                            Page {table.getState().pagination.pageIndex + 1} of{" "}
                            {table.getPageCount()}
                        </div>
                        <div className="ml-auto flex items-center gap-2 lg:ml-0">
                            <Button
                                variant="outline"
                                className="hidden h-10 w-10 p-0 lg:flex touch-manipulation"
                                onClick={() => table.setPageIndex(0)}
                                disabled={!table.getCanPreviousPage()}
                            >
                                <span className="sr-only">
                                    Go to first page
                                </span>
                                <IconChevronsLeft />
                            </Button>
                            <Button
                                variant="outline"
                                className="h-10 w-10 touch-manipulation"
                                size="icon"
                                onClick={() => table.previousPage()}
                                disabled={!table.getCanPreviousPage()}
                            >
                                <span className="sr-only">
                                    Go to previous page
                                </span>
                                <IconChevronLeft />
                            </Button>
                            <Button
                                variant="outline"
                                className="h-10 w-10 touch-manipulation"
                                size="icon"
                                onClick={() => table.nextPage()}
                                disabled={!table.getCanNextPage()}
                            >
                                <span className="sr-only">Go to next page</span>
                                <IconChevronRight />
                            </Button>
                            <Button
                                variant="outline"
                                className="hidden h-10 w-10 lg:flex touch-manipulation"
                                size="icon"
                                onClick={() =>
                                    table.setPageIndex(table.getPageCount() - 1)
                                }
                                disabled={!table.getCanNextPage()}
                            >
                                <span className="sr-only">Go to last page</span>
                                <IconChevronsRight />
                            </Button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}
