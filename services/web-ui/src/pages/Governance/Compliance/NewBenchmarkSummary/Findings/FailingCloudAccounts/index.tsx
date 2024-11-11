import { useAtomValue } from 'jotai'
import { Card, Flex, Text } from '@tremor/react'
import { useEffect, useState } from 'react'
import { ICellRendererParams, RowClickedEvent } from 'ag-grid-community'
import { useComplianceApiV1FindingsTopDetail } from '../../../../../../api/compliance.gen'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus,
    SourceType,
    TypesFindingSeverity,
} from '../../../../../../api/api'
import Table, { IColumn } from '../../../../../../components/Table'
import { topConnections } from '../../../../Controls/ControlSummary/Tabs/ImpactedAccounts'
import { getConnectorIcon } from '../../../../../../components/Cards/ConnectorCard'
import CloudAccountDetail from './Detail'
import { isDemoAtom } from '../../../../../../store'

const cloudAccountColumns = (isDemo: boolean) => {
    const temp: IColumn<any, any>[] = [
        {
            field: 'providerConnectionName',
            headerName: 'Account name',
            resizable: true,
            type: 'string',
            sortable: true,
            filter: true,
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
            headerName: 'Findings',
            field: 'count',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            width: 150,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                    className="h-full"
                >
                    <Text className="text-gray-800">
                        {param.value || 0} issues
                    </Text>
                    <Text>
                        {(param.data.totalCount || 0) - (param.value || 0)}{' '}
                        passed
                    </Text>
                </Flex>
            ),
        },
        {
            headerName: 'Resources',
            field: 'resourceCount',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            width: 150,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                    className="h-full"
                >
                    <Text className="text-gray-800">
                        {param.value || 0} issues
                    </Text>
                    <Text>
                        {(param.data.resourceTotalCount || 0) -
                            (param.value || 0)}{' '}
                        passed
                    </Text>
                </Flex>
            ),
        },
        {
            headerName: 'Controls',
            field: 'controlCount',
            type: 'number',
            sortable: true,
            filter: true,
            resizable: true,
            width: 150,
            cellRenderer: (param: ICellRendererParams) => (
                <Flex
                    flexDirection="col"
                    alignItems="start"
                    justifyContent="center"
                    className="h-full"
                >
                    <Text className="text-gray-800">
                        {param.value || 0} issues
                    </Text>
                    <Text>
                        {(param.data.controlTotalCount || 0) -
                            (param.value || 0)}{' '}
                        passed
                    </Text>
                </Flex>
            ),
        },
    ]
    return temp
}

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

export default function FailingCloudAccounts({ query }: ICount) {
    const [account, setAccount] = useState<any>(undefined)
    const [open, setOpen] = useState(false)
    const isDemo = useAtomValue(isDemoAtom)

    const topQuery = {
        connector: query.connector.length ? [query.connector] : [],
        connectionId: query.connectionID,
        benchmarkId: query.benchmarkID,
    }

    const { response: accounts, isLoading: accountsLoading } =
        useComplianceApiV1FindingsTopDetail('connectionID', 10000, topQuery)

    return (
        <>
            <Table
                id="impacted_accounts"
                columns={cloudAccountColumns(isDemo)}
                rowData={topConnections(accounts)}
                loading={accountsLoading}
                onGridReady={(e) => {
                    if (accountsLoading) {
                        e.api.showLoadingOverlay()
                    }
                }}
                onCellClicked={(event: RowClickedEvent) => {
                    setAccount(event.data)
                    setOpen(true)
                }}
                rowHeight="lg"
            />
            <CloudAccountDetail
                account={account}
                open={open}
                onClose={() => setOpen(false)}
            />
        </>
    )
}
