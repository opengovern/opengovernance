import {
    CellClickedEvent,
    ColDef,
    ColGroupDef,
    ColumnRowGroupChangedEvent,
    GridOptions,
    GridReadyEvent,
    IAggFunc,
    ICellRendererParams,
    IServerSideDatasource,
    NestedFieldPaths,
    RowClickedEvent,
    ValueFormatterFunc,
} from 'ag-grid-community'
import { AgGridReact } from 'ag-grid-react'

import 'ag-grid-community/styles/ag-grid.css'
import 'ag-grid-community/styles/ag-theme-alpine.css'
import { ReactNode, useEffect, useRef, useState } from 'react'
import { Button, Flex, Text, TextInput, Title } from '@tremor/react'
import { ArrowDownTrayIcon } from '@heroicons/react/20/solid'
import {
    exactPriceDisplay,
    numberGroupedDisplay,
} from '../../utilities/numericDisplay'
import { agGridDateComparator } from '../../utilities/dateComparator'
import { getConnectorIcon } from '../Cards/ConnectorCard'
import { dateDisplay, dateTimeDisplay } from '../../utilities/dateDisplay'
import Spinner from '../Spinner'

export interface IColumn<TData, TValue> {
    type:
        | 'string'
        | 'number'
        | 'price'
        | 'date'
        | 'datetime'
        | 'connector'
        | 'custom'
    field?: NestedFieldPaths<TData, any>
    width?: number
    cellStyle?: any
    headerName?: string
    cellDataType?: boolean | string
    valueFormatter?: string | ValueFormatterFunc<TData, TValue>
    comparator?: any
    cellRenderer?: any
    cellRendererParams?: any
    rowGroup?: boolean
    enableRowGroup?: boolean
    pinned?: boolean
    aggFunc?: string | IAggFunc<TData, TValue> | null
    suppressMenu?: boolean
    floatingFilter?: boolean
    pivot?: boolean
    hide?: boolean
    filter?: any
    filterParams?: any
    sortable?: boolean
    resizable?: boolean
    flex?: number
    isBold?: boolean
}

interface IProps<TData, TValue> {
    id: string
    columns: IColumn<TData, TValue>[]
    rowData?: TData[] | undefined
    pinnedRow?: TData[] | undefined
    serverSideDatasource?: IServerSideDatasource | undefined
    onGridReady?: (event: GridReadyEvent<TData>) => void
    onCellClicked?: (event: CellClickedEvent<TData>) => void
    onRowClicked?: (event: RowClickedEvent<TData>) => void
    onColumnRowGroupChanged?: (event: ColumnRowGroupChangedEvent<TData>) => void
    onSortChange?: () => void
    downloadable?: boolean
    title?: string
    children?: ReactNode
    options?: GridOptions
    loading?: boolean
    fullWidth?: boolean
    fullHeight?: boolean
    rowHeight?: 'md' | 'lg' | 'xl'
    quickFilter?: string
    masterDetail?: boolean
    detailCellRenderer?: any
    detailCellRendererParams?: any
    detailRowHeight?: number
}

export default function Table<TData = any, TValue = any>({
    id,
    columns,
    rowData,
    pinnedRow,
    serverSideDatasource,
    onGridReady,
    onCellClicked,
    onRowClicked,
    onColumnRowGroupChanged,
    onSortChange,
    downloadable = false,
    fullWidth = false,
    fullHeight = false,
    title,
    children,
    options,
    loading,
    rowHeight = 'md',
    quickFilter,
    masterDetail,
    detailCellRenderer,
    detailCellRendererParams,
    detailRowHeight,
}: IProps<TData, TValue>) {
    const gridRef = useRef<AgGridReact>(null)
    const visibility = useRef<Map<string, boolean> | undefined>(undefined)

    if (visibility.current === undefined) {
        visibility.current = new Map()
        const columnVisibility = localStorage.getItem(
            `table_${id}_columns_visibility`
        )
        if (columnVisibility) {
            const v = JSON.parse(columnVisibility)
            if (typeof v === 'object') {
                Object.entries(v).forEach((vi) => {
                    visibility.current?.set(vi[0], Boolean(vi[1]))
                })
            }
        }
    }

    useEffect(() => {
        if (loading) {
            gridRef.current?.api?.showLoadingOverlay()
        } else {
            gridRef.current?.api?.hideOverlay()
        }
    }, [loading])

    const saveVisibility = () => {
        if (visibility.current) {
            const o = Object.fromEntries(visibility.current.entries())
            localStorage.setItem(
                `table_${id}_columns_visibility`,
                JSON.stringify(o)
            )
        }
    }

    const buildColumnDef = () => {
        return columns?.map((item) => {
            const v: ColDef<TData> | ColGroupDef<TData> | any = {
                field: item.field,
                headerName: item.headerName,
                filter: item.filter,
                filterParams: item.filterParams,
                width: item.width,
                sortable: item.sortable === undefined ? true : item.sortable,
                resizable: item.resizable === undefined ? true : item.resizable,
                rowGroup: item.rowGroup || false,
                enableRowGroup: item.enableRowGroup || false,
                hide: item.hide || false,
                cellRenderer: item.cellRenderer,
                cellRendererParams: item.cellRendererParams,
                flex: item.width ? 0 : item.flex || 1,
                pinned: item.pinned || false,
                aggFunc: item.aggFunc,
                suppressMenu: item.suppressMenu || false,
                floatingFilter: item.floatingFilter || false,
                pivot: item.pivot || false,
                valueFormatter: item.valueFormatter,
                comparator: item.comparator,
                cellStyle: item.cellStyle,
            }

            if (
                item.field &&
                visibility.current?.get(item.field || '') !== undefined
            ) {
                v.hide = !visibility.current.get(item.field || '')
            }

            // v.cellStyle = {
            //     display: 'flex',
            //     'align-content': 'center',
            // }

            if (item.type === 'string') {
                v.cellDataType = 'text'
                if (
                    item.cellRenderer === undefined &&
                    item.valueFormatter === undefined
                ) {
                    v.cellRenderer = (params: ICellRendererParams<TData>) => (
                        <Flex
                            className={`${item.isBold ? ' text-gray-900' : ''}`}
                        >
                            {params.value}
                        </Flex>
                    )
                }
            }

            if (item.type === 'price') {
                v.filter = 'agNumberColumnFilter'
                v.cellDataType = 'text'
                v.valueFormatter = (param: any) => {
                    return (
                        exactPriceDisplay(String(param.value)) ||
                        'Not available'
                    )
                }
            } else if (item.type === 'number') {
                v.filter = 'agNumberColumnFilter'
                v.cellDataType = 'number'
                v.valueFormatter = (param: any) => {
                    return param.value || param.value === 0
                        ? numberGroupedDisplay(param.value)
                        : 'Not available'
                }
            } else if (item.type === 'date') {
                v.filter = 'agDateColumnFilter'
                v.filterParams = {
                    comparator: agGridDateComparator,
                }
                v.valueFormatter = (param: any) => {
                    if (param.value) {
                        let value = ''
                        if (!Number.isNaN(Number(param.value))) {
                            value = dateDisplay(
                                Number(param.value) > 16000000000
                                    ? Number(param.value)
                                    : Number(param.value) * 1000
                            )
                        } else {
                            value = dateDisplay(param.value)
                        }
                        return value
                    }
                    return 'Not available'
                }
            } else if (item.type === 'datetime') {
                v.filter = 'agDateColumnFilter'
                v.filterParams = {
                    comparator: agGridDateComparator,
                }
                v.valueFormatter = (param: any) => {
                    if (param.value) {
                        let value = ''
                        if (!Number.isNaN(Number(param.value))) {
                            value = dateTimeDisplay(
                                Number(param.value) > 16000000000
                                    ? Number(param.value)
                                    : Number(param.value) * 1000
                            )
                        } else {
                            value = dateTimeDisplay(param.value)
                        }
                        return value
                    }
                    return 'Not available'
                }
            } else if (item.type === 'connector') {
                v.width = 50
                v.cellStyle = { padding: 0 }
                v.cellRenderer = (params: ICellRendererParams<TData>) =>
                    getConnectorIcon(
                        params.value,
                        '!w-full !h-full justify-center'
                    )
            }
            return v
        })
    }

    useEffect(() => {
        gridRef.current?.api?.setGridOption('columnDefs', buildColumnDef())
    }, [columns])
    useEffect(() => {
        if (pinnedRow) {
            gridRef.current?.api?.setGridOption('pinnedTopRowData', pinnedRow)
        }
    }, [pinnedRow])
    useEffect(() => {
        if (rowData) {
            gridRef.current?.api?.setGridOption('rowData', rowData || [])
        }
    }, [rowData])
    useEffect(() => {
        if (serverSideDatasource) {
            gridRef.current?.api?.setGridOption(
                'serverSideDatasource',
                serverSideDatasource
            )
        }
    }, [serverSideDatasource])

    const gridOptions: GridOptions = {
        rowModelType: serverSideDatasource ? 'serverSide' : 'clientSide',
        columnDefs: buildColumnDef(),
        suppressContextMenu: !!serverSideDatasource,
        ...(rowData && { rowData: rowData || [] }),
        ...(serverSideDatasource && {
            // serverSideDatasource,
            cacheBlockSize: 25,
            maxBlocksInCache: 10000,
            // maxConcurrentDatasourceRequests: -1,
        }),
        pagination: true,
        paginationPageSize: 25,
        masterDetail,
        detailCellRenderer,
        detailCellRendererParams,
        detailRowHeight,
        rowSelection: 'multiple',
        suppressExcelExport: true,
        alwaysShowHorizontalScroll: true,
        suppressCellFocus: true,
        suppressMenuHide: true,
        animateRows: false,
        quickFilterText: quickFilter,
        getRowHeight: () => {
            if (rowHeight === 'md') {
                return 50
            }
            if (rowHeight === 'lg') {
                return 64
            }
            return 80
        },
        onGridReady: (e) => {
            if (onGridReady) {
                onGridReady(e)
            }
        },
        onSortChanged: (e) => {
            if (serverSideDatasource) {
                e.api.paginationGoToPage(0)
            }
            if (onSortChange) {
                onSortChange()
            }
        },
        onCellClicked,
        onRowClicked,
        onColumnRowGroupChanged,
        onColumnVisible: (e) => {
            if (e.column?.getId() && e.visible !== undefined) {
                visibility.current?.set(e.column?.getId(), e.visible)
                saveVisibility()
            }
        },
        sideBar: {
            toolPanels: [
                {
                    id: 'columns',
                    labelDefault: 'Columns',
                    labelKey: 'columns',
                    iconKey: 'columns',
                    toolPanel: 'agColumnsToolPanel',
                },
                // {
                //     id: 'filters',
                //     labelDefault: 'Table Filters',
                //     labelKey: 'filters',
                //     iconKey: 'filter',
                //     toolPanel: 'agFiltersToolPanel',
                // },
            ],
            defaultToolPanel: '',
        },
        ...options,
    }

    useEffect(() => {
        gridRef.current?.api?.updateGridOptions(gridOptions)
    }, [quickFilter])

    return (
        <Flex
            flexDirection="col"
            className={`w-full ${fullHeight ? 'h-full' : ''}`}
        >
            <Flex
                className={
                    !!title?.length || downloadable || children ? 'mb-3' : ''
                }
            >
                {!!title?.length && (
                    <Title className="font-semibold min-w-fit">{title}</Title>
                )}
                <Flex
                    flexDirection={fullWidth ? 'row-reverse' : 'row'}
                    alignItems={fullWidth ? 'start' : 'center'}
                    className={`${fullWidth ? '' : 'w-fit'} gap-3`}
                >
                    {children}
                    {downloadable && (
                        <Button
                            variant="secondary"
                            onClick={() => {
                                gridRef.current?.api.exportDataAsCsv()
                            }}
                            icon={ArrowDownTrayIcon}
                        >
                            Download
                        </Button>
                    )}
                </Flex>
            </Flex>

            <div
                className={`w-full relative overflow-hidden ${
                    localStorage.theme === 'dark'
                        ? 'ag-theme-alpine-dark'
                        : 'ag-theme-alpine'
                } ${fullHeight ? 'h-full' : ''}`}
            >
                {loading && (
                    <Flex
                        justifyContent="center"
                        alignItems="center"
                        className="top-[50px] right-[32px] z-10 backdrop-blur h-full absolute"
                    >
                        <Spinner />
                    </Flex>
                )}
                <AgGridReact
                    ref={gridRef}
                    domLayout="autoHeight"
                    gridOptions={gridOptions}
                    // rowData={rowData}
                />
            </div>
        </Flex>
    )
}
