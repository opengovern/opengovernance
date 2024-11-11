import { Card, Flex, Text } from '@tremor/react'
import {
    ICellRendererParams,
    RowClickedEvent,
    ValueFormatterParams,
    IServerSideGetRowsParams,
} from 'ag-grid-community'
import { useMemo, useState } from 'react'
import { useAtomValue, useSetAtom } from 'jotai/index'
import {
    Api,
    GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus,
    GithubComKaytuIoKaytuEnginePkgComplianceApiResourceFinding,
    SourceType,
    TypesFindingSeverity,
} from '../../../../../../api/api'
import AxiosAPI from '../../../../../../api/ApiConfig'
import { isDemoAtom, notificationAtom } from '../../../../../../store'
import Table, { IColumn } from '../../../../../../components/Table'
import { dateTimeDisplay } from '../../../../../../utilities/dateDisplay'
import { getConnectorIcon } from '../../../../../../components/Cards/ConnectorCard'
import ResourceFindingDetail from '../ResourceFindingDetail'

const columns = (isDemo: boolean) => {
    const temp: IColumn<any, any>[] = [
        {
            field: 'resourceName',
            headerName: 'Resource name',
            hide: false,
            type: 'string',
            enableRowGroup: true,
            sortable: false,
            filter: true,
            resizable: true,
            flex: 1,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                    className="h-full"
                >
                    <Text className="text-gray-800">{param.value}</Text>
                    <Text className={isDemo ? 'blur-sm' : ''}>
                        {param.data.kaytuResourceID}
                    </Text>
                </Flex>
            ),
        },
        {
            field: 'resourceType',
            headerName: 'Resource type',
            type: 'string',
            enableRowGroup: true,
            sortable: true,
            hide: false,
            filter: true,
            resizable: true,
            flex: 1,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                >
                    <Text className="text-gray-800">{param.value}</Text>
                    <Text>{param.data.resourceTypeLabel}</Text>
                </Flex>
            ),
        },
        {
            field: 'resourceLocation',
            headerName: 'Resource location',
            type: 'string',
            enableRowGroup: true,
            sortable: false,
            hide: true,
            filter: true,
            resizable: true,
            flex: 1,
        },
        {
            field: 'providerConnectionName',
            headerName: 'Cloud account',
            sortable: false,
            filter: true,
            hide: false,
            enableRowGroup: true,
            type: 'string',
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    justifyContent="start"
                    className={`h-full gap-3 group relative ${
                        isDemo ? 'blur-sm' : ''
                    }`}
                >
                    {getConnectorIcon(param.data.connector)}
                    <Flex flexDirection="col" alignItems="start">
                        <Text className="text-gray-800">{param.value}</Text>
                        <Text>{param.data.providerConnectionID}</Text>
                    </Flex>
                    <Card className="cursor-pointer absolute w-fit h-fit z-40 right-1 scale-0 transition-all py-1 px-4 group-hover:scale-100">
                        <Text color="blue">Open</Text>
                    </Card>
                </Flex>
            ),
        },
        {
            field: 'totalCount',
            headerName: 'Findings',
            type: 'number',
            hide: false,
            enableRowGroup: true,
            sortable: false,
            filter: true,
            resizable: true,
            width: 140,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                    className="h-full"
                >
                    <Text className="text-gray-800">{`${param.value} issues`}</Text>
                    <Text>{`${
                        param.value - param.data.failedCount
                    } passed`}</Text>
                </Flex>
            ),
        },
        {
            field: 'evaluatedAt',
            headerName: 'Last checked',
            type: 'datetime',
            sortable: false,
            filter: true,
            resizable: true,
            flex: 1,
            valueFormatter: (param: ValueFormatterParams) => {
                return param.value ? dateTimeDisplay(param.value) : ''
            },
            hide: true,
        },
    ]
    return temp
}

let sortKey: any[] = []

interface ICount {
    query: {
        connector: SourceType
        conformanceStatus:
            | GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus[]
            | undefined
        severity: TypesFindingSeverity[] | undefined
        connectionID: string[] | undefined
        controlID: string[] | undefined
        benchmarkID: string[] | undefined
        resourceTypeID: string[] | undefined
        lifecycle: boolean[] | undefined
    }
}

export default function ResourcesWithFailure({ query }: ICount) {
    const setNotification = useSetAtom(notificationAtom)

    const [open, setOpen] = useState(false)
    const [finding, setFinding] = useState<
        GithubComKaytuIoKaytuEnginePkgComplianceApiResourceFinding | undefined
    >(undefined)

    const isDemo = useAtomValue(isDemoAtom)

    const ssr = () => {
        return {
            getRows: (
                params: IServerSideGetRowsParams<GithubComKaytuIoKaytuEnginePkgComplianceApiResourceFinding>
            ) => {
                const api = new Api()
                api.instance = AxiosAPI
                api.compliance
                    .apiV1ResourceFindingsCreate({
                        filters: {
                            connector: query.connector.length
                                ? [query.connector]
                                : [],
                            controlID: query.controlID,
                            connectionID: query.connectionID,
                            benchmarkID: query.benchmarkID,
                            severity: query.severity,
                            resourceTypeID: query.resourceTypeID,
                            conformanceStatus: query.conformanceStatus,
                        },
                        sort: params.request.sortModel.length
                            ? [
                                  {
                                      [params.request.sortModel[0].colId]:
                                          params.request.sortModel[0].sort,
                                  },
                              ]
                            : [],
                        limit: 100,
                        afterSortKey:
                            params.request.startRow === 0 || sortKey.length < 1
                                ? []
                                : sortKey,
                    })
                    .then((resp) => {
                        params.success({
                            rowData: resp.data.resourceFindings || [],
                            rowCount: resp.data.totalCount || 0,
                        })
                        sortKey =
                            resp.data?.resourceFindings?.at(
                                (resp.data.resourceFindings?.length || 0) - 1
                            )?.sortKey || []
                    })
                    .catch((err) => {
                        params.fail()
                    })
            },
        }
    }

    const serverSideRows = useMemo(() => ssr(), [query])

    return (
        <>
            <Table
                fullWidth
                id="compliance_findings"
                columns={columns(isDemo)}
                onCellClicked={(
                    event: RowClickedEvent<GithubComKaytuIoKaytuEnginePkgComplianceApiResourceFinding>
                ) => {
                    if (
                        event.data?.kaytuResourceID &&
                        event.data?.kaytuResourceID.length > 0
                    ) {
                        setFinding(event.data)
                        setOpen(true)
                    } else {
                        setNotification({
                            text: 'Detail for this finding is currently not available',
                            type: 'warning',
                        })
                    }
                }}
                onSortChange={() => {
                    sortKey = []
                }}
                serverSideDatasource={serverSideRows}
                options={{
                    rowModelType: 'serverSide',
                    serverSideDatasource: serverSideRows,
                }}
                rowHeight="lg"
            />
            <ResourceFindingDetail
                // type="resource"
                resourceFinding={finding}
                open={open}
                showOnlyOneControl={false}
                onClose={() => setOpen(false)}
                onRefresh={() => window.location.reload()}
            />
        </>
    )
}
