import { Col, Grid } from '@tremor/react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import {
    useInventoryApiV2AnalyticsSpendMetricList,
    useInventoryApiV2AnalyticsSpendTableList,
} from '../../../api/inventory.gen'
import { SpendChart } from '../../../components/Spend/Chart'
import { toErrorMessage } from '../../../types/apierror'
import {
    ChartLayout,
    Granularity,
} from '../../../components/Spend/Chart/Selectors'
import AccountTable from './AccountTable'
import {
    GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint,
    GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow,
} from '../../../api/api'
import TopHeader from '../../../components/Layout/Header'
import {
    defaultSpendTime,
    useFilterState,
    useURLParam,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export const accountTrend = (
    responseChart: GithubComKaytuIoKaytuEnginePkgInventoryApiSpendTableRow[],
    chartLayout: ChartLayout
) => {
    return responseChart
        ?.flatMap((item) =>
            Object.entries(item.costValue || {}).map((entry) => {
                return {
                    accountID: item.dimensionId || item.accountID || '',
                    accountName: item.dimensionName,
                    connector: item.connector,
                    date: entry[0],
                    cost: entry[1],
                }
            })
        )
        .reduce<GithubComKaytuIoKaytuEnginePkgInventoryApiCostTrendDatapoint[]>(
            (prev, curr) => {
                const stacked = {
                    cost: curr.cost,
                    metricID:
                        chartLayout === 'accounts'
                            ? curr.accountID
                            : curr.connector,
                    metricName:
                        chartLayout === 'accounts'
                            ? curr.accountName
                            : curr.connector,
                }
                const exists =
                    prev.filter((p) => p.date === curr.date).length > 0
                if (exists) {
                    return prev.map((p) => {
                        if (p.date === curr.date) {
                            return {
                                cost: (p.cost || 0) + curr.cost,
                                costStacked: [
                                    ...(p.costStacked || []),
                                    stacked,
                                ],
                                date: curr.date,
                            }
                        }
                        return p
                    })
                }
                return [
                    ...prev,
                    {
                        cost: curr.cost,
                        costStacked: [stacked],
                        date: curr.date,
                    },
                ]
            },
            []
        )
}

export function SpendAccounts() {
    const { ws } = useParams()
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultSpendTime(ws || '')
    )
    const { value: selectedConnections } = useFilterState()
    const [chartGranularity, setChartGranularity] =
        useState<Granularity>('daily')
    const [tableGranularity, setTableGranularity] =
        useState<Granularity>('daily')

    const query: {
        pageSize: number
        pageNumber: number
        sortBy: 'cost' | undefined
        endTime: number
        startTime: number
        connectionId: string[]
        connector?: ('AWS' | 'Azure')[] | undefined
    } = {
        ...(selectedConnections.provider !== '' && {
            connector: [selectedConnections.provider],
        }),
        ...(selectedConnections.connections && {
            connectionId: selectedConnections.connections,
        }),
        ...(selectedConnections.connectionGroup && {
            connectionGroup: selectedConnections.connectionGroup,
        }),
        ...(activeTimeRange.start && {
            startTime: activeTimeRange.start.unix(),
        }),
        ...(activeTimeRange.end && {
            endTime: activeTimeRange.end.unix(),
        }),
        pageSize: 5,
        pageNumber: 1,
        sortBy: 'cost',
    }

    const duration =
        activeTimeRange.end.diff(activeTimeRange.start, 'second') + 1
    const prevTimeRange = {
        start: activeTimeRange.start.add(-duration, 'second'),
        end: activeTimeRange.end.add(-duration, 'second'),
    }
    const prevQuery = {
        ...query,
        ...(activeTimeRange.start && {
            startTime: prevTimeRange.start.unix(),
        }),
        ...(activeTimeRange.end && {
            endTime: prevTimeRange.end.unix(),
        }),
    }

    const {
        response: serviceCostResponse,
        isLoading: serviceCostLoading,
        error: serviceCostErr,
        sendNow: serviceCostRefresh,
    } = useInventoryApiV2AnalyticsSpendMetricList(query)
    const {
        response: servicePrevCostResponse,
        isLoading: servicePrevCostLoading,
        error: servicePrevCostErr,
        sendNow: serviceCostPrevRefresh,
    } = useInventoryApiV2AnalyticsSpendMetricList(prevQuery)

    const { response: responseChart, isLoading: isLoadingChart } =
        useInventoryApiV2AnalyticsSpendTableList({
            startTime: activeTimeRange.start.unix(),
            endTime: activeTimeRange.end.unix(),
            dimension: 'connection',
            granularity: chartGranularity,
            connector:
                selectedConnections.provider === ''
                    ? []
                    : [selectedConnections.provider],
            connectionId: selectedConnections.connections,
            connectionGroup: selectedConnections.connectionGroup,
        })

    const { response, isLoading } = useInventoryApiV2AnalyticsSpendTableList({
        startTime: activeTimeRange.start.unix(),
        endTime: activeTimeRange.end.unix(),
        dimension: 'connection',
        granularity: tableGranularity,
        connector:
            selectedConnections.provider === ''
                ? []
                : [selectedConnections.provider],
        connectionId: selectedConnections.connections,
        connectionGroup: selectedConnections.connectionGroup,
    })
    const { response: responsePrev, isLoading: prevIsLoading } =
        useInventoryApiV2AnalyticsSpendTableList({
            startTime: prevTimeRange.start.unix(),
            endTime: prevTimeRange.end.unix(),
            dimension: 'connection',
            granularity: tableGranularity,
            connector:
                selectedConnections.provider === ''
                    ? []
                    : [selectedConnections.provider],
            connectionId: selectedConnections.connections,
            connectionGroup: selectedConnections.connectionGroup,
        })

    const [chartLayout, setChartLayout] = useURLParam<ChartLayout>(
        'show',
        'accounts'
    )
    return (
        <>
            <TopHeader
                supportedFilters={['Date', 'Cloud Account', 'Connector']}
                initialFilters={['Date']}
                datePickerDefault={defaultSpendTime(ws || '')}
            />
            <Grid numItems={3} className="w-full gap-4">
                <Col numColSpan={3}>
                    <SpendChart
                        costTrend={
                            accountTrend(responseChart || [], chartLayout) || []
                        }
                        title="Total spend"
                        timeRange={activeTimeRange}
                        timeRangePrev={prevTimeRange}
                        total={serviceCostResponse?.total_cost || 0}
                        totalPrev={servicePrevCostResponse?.total_cost || 0}
                        chartLayout={chartLayout}
                        setChartLayout={setChartLayout}
                        validChartLayouts={['total', 'provider', 'accounts']}
                        // noStackedChart
                        isLoading={
                            serviceCostLoading ||
                            servicePrevCostLoading ||
                            isLoadingChart
                        }
                        error={toErrorMessage(
                            serviceCostErr,
                            servicePrevCostErr
                        )}
                        onRefresh={() => {
                            serviceCostPrevRefresh()
                            serviceCostRefresh()
                        }}
                        onGranularityChanged={setChartGranularity}
                    />
                </Col>
                <Col numColSpan={3} className="mt-6">
                    <AccountTable
                        isLoading={isLoading || prevIsLoading}
                        response={response}
                        responsePrev={responsePrev}
                        onGranularityChange={setTableGranularity}
                        selectedGranularity={tableGranularity}
                        timeRange={activeTimeRange}
                        prevTimeRange={prevTimeRange}
                    />
                </Col>
            </Grid>
        </>
    )
}
