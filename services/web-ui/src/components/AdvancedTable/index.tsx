import {
    CellClickedEvent,
    ColDef,
    ColDefField,
    ColGroupDef,
    ColumnGroupShowType,
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
import 'ag-grid-enterprise'
import 'ag-grid-community/styles/ag-grid.css'
import 'ag-grid-community/styles/ag-theme-alpine.css'
import {
    ReactNode,
    useEffect,
    useRef,
    Dispatch,
    SetStateAction,
    useState,
} from 'react'
import { Button, Flex, Title } from '@tremor/react'
import { ArrowDownTrayIcon } from '@heroicons/react/20/solid'
import {
    exactPriceDisplay,
    numberGroupedDisplay,
} from '../../utilities/numericDisplay'
import { agGridDateComparator } from '../../utilities/dateComparator'
import { getConnectorIcon } from '../Cards/ConnectorCard'
import { dateDisplay, dateTimeDisplay } from '../../utilities/dateDisplay'
import Spinner from '../Spinner'
import TableGranularityControl from './TableGranularityControl'
import FilterTabs from './FilterTabs'

type FilterTab = {
    type: number
    icon: React.ForwardRefExoticComponent<
        Omit<React.SVGProps<SVGSVGElement>, 'ref'> & {
            title?: string | undefined
            titleId?: string | undefined
        } & React.RefAttributes<SVGSVGElement>
    >
    name: string
    function: () => void
}[]

type MSort = {
    sortCol: string
    sortType: 'asc' | 'desc' | null
}

export interface IColumn<TData, TValue> {
    children?: IColumn<TData, TValue>[]
    type:
        | 'string'
        | 'number'
        | 'price'
        | 'date'
        | 'datetime'
        | 'connector'
        | 'parent'
        | 'custom'
    field?: ColDefField<TData, TValue>
    width?: number
    cellStyle?: any
    headerName?: string
    cellDataType?: boolean | string
    valueFormatter?: string | ValueFormatterFunc<TData, TValue>
    comparator?: any
    cellRenderer?: any
    rowGroup?: boolean
    enableRowGroup?: boolean
    pinned?: boolean
    aggFunc?: string | IAggFunc<TData, TValue> | null
    suppressMenu?: boolean
    floatingFilter?: boolean
    pivot?: boolean
    hide?: boolean
    filter?: boolean
    filterParams?: any
    sortable?: boolean
    resizable?: boolean
    flex?: number
    wrapText?: boolean
    autoHeight?: boolean
    wrapHeaderText?: boolean
    autoHeaderHeight?: boolean
    columnGroupShow?: ColumnGroupShowType
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
    onSortChange?: () => void
    downloadable?: boolean
    title?: string
    children?: ReactNode
    options?: GridOptions
    loading?: boolean
    fullWidth?: boolean
    fullHeight?: boolean
    selectedGranularity?: 'daily' | 'monthly'
    onGranularityChange?: Dispatch<SetStateAction<'monthly' | 'daily'>>
    manualSort?: MSort
    manualGrouping?: string
    filterTabs?: FilterTab
    tabIdx: number
    setTabIdx: (v: number) => void
    ref?: React.MutableRefObject<any>
}

export default function AdvancedTable<TData = any, TValue = any>({
    id,
    columns,
    rowData,
    pinnedRow,
    serverSideDatasource,
    onGridReady,
    onCellClicked,
    onRowClicked,
    onSortChange,
    downloadable = false,
    fullWidth = false,
    fullHeight = false,
    title,
    children,
    options,
    loading,
    selectedGranularity,
    onGranularityChange,
    manualSort,
    manualGrouping,
    filterTabs,
    tabIdx,
    setTabIdx,
    ref,
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

    if (manualSort !== undefined) {
        gridRef.current?.api.applyColumnState({
            defaultState: { sort: null },
        })
        gridRef.current?.api.applyColumnState({
            state: [{ colId: manualSort.sortCol, sort: manualSort.sortType }],
        })
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

    const convert = (item: IColumn<TData, TValue>) => {
        if (item.children !== undefined) {
            const v: ColGroupDef<TData> = {
                headerName: item.headerName,
                headerClass: 'font-bold text-normal text-gray-900',
                marryChildren: true,
                children: item.children.map((i) => convert(i)),
            }
            return v
        }

        const v: ColDef<TData> = {
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
            flex: item.width ? 0 : item.flex || 1,
            pinned: item.pinned || false,
            aggFunc: item.aggFunc,
            suppressMenu: item.suppressMenu || false,
            floatingFilter: item.floatingFilter || false,
            pivot: item.pivot || false,
            valueFormatter: item.valueFormatter,
            comparator: item.comparator,
            wrapText: item.wrapText,
            autoHeight: item.autoHeight,
            wrapHeaderText: item.wrapHeaderText,
            autoHeaderHeight: item.autoHeaderHeight,
            columnGroupShow: item.columnGroupShow,
        }

        if (
            item.field &&
            visibility.current?.get(item.field || '') !== undefined
        ) {
            v.hide = !visibility.current.get(item.field || '')
        }

        if (item.field === manualGrouping) {
            v.enableRowGroup = true
            v.rowGroup = true
        }

        if (item.type === 'price') {
            v.filter = 'agNumberColumnFilter'
            v.cellDataType = 'text'
            v.valueFormatter = (param: any) => {
                return exactPriceDisplay(String(param.value)) || 'Not available'
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
                getConnectorIcon(params.value)
        }
        return v
    }

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
        columnDefs: columns?.map((item) => convert(item)),
        ...(rowData && { rowData: rowData || [] }),
        ...(serverSideDatasource && {
            // serverSideDatasource,
            cacheBlockSize: 25,
            maxBlocksInCache: 10000,
            // maxConcurrentDatasourceRequests: -1,
        }),
        pagination: true,
        paginationPageSize: 10,
        rowSelection: 'multiple',
        pinnedTopRowData: pinnedRow,
        suppressExcelExport: true,
        alwaysShowHorizontalScroll: true,
        animateRows: false,
        scrollbarWidth: 8,
        getRowHeight: () => 50,
        onGridReady: (e) => {
            if (onGridReady) {
                onGridReady(e)

                if (manualSort !== undefined) {
                    e.api.applyColumnState({
                        defaultState: { sort: null },
                    })
                    e.api.applyColumnState({
                        state: [
                            {
                                colId: manualSort.sortCol,
                                sort: manualSort.sortType,
                            },
                        ],
                    })
                }
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

    // useEffect(() => {
    //     gridRef.current?.api?.updateGridOptions(gridOptions)
    // }, [columns])

    return (
        <Flex
            ref={ref}
            flexDirection="col"
            className={`w-full ${fullHeight ? 'h-full' : ''}`}
        >
            <Flex
                className={
                    !!title?.length || downloadable || children ? 'mb-3' : ''
                }
            >
                {!!title?.length && (
                    <Title className="font-bold min-w-fit">{title}</Title>
                )}
                <Flex
                    flexDirection={fullWidth ? 'row-reverse' : 'row'}
                    alignItems={fullWidth ? 'start' : 'center'}
                    className={`${fullWidth ? '' : 'w-fit'} gap-3`}
                >
                    {selectedGranularity && onGranularityChange && (
                        <TableGranularityControl
                            selectedGranularity={selectedGranularity}
                            onGranularityChange={onGranularityChange}
                        />
                    )}
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

            {filterTabs !== undefined && (
                <Flex className="m-4">
                    <FilterTabs
                        tabs={filterTabs}
                        tab={tabIdx}
                        setTab={setTabIdx}
                    />
                </Flex>
            )}

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
