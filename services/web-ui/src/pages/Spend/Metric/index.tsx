import { Col, Grid } from '@tremor/react'
import { useState } from 'react'
import { useParams } from 'react-router-dom'
import {
    useInventoryApiV2AnalyticsSpendMetricList,
    useInventoryApiV2AnalyticsSpendTableList,
    useInventoryApiV2AnalyticsSpendTrendList,
} from '../../../api/inventory.gen'
import { SpendChart } from '../../../components/Spend/Chart'
import { toErrorMessage } from '../../../types/apierror'
import {
    ChartLayout,
    Granularity,
} from '../../../components/Spend/Chart/Selectors'
import MetricTable from './MetricTable'
import TopHeader from '../../../components/Layout/Header'
import {
    defaultSpendTime,
    useFilterState,
    useURLParam,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export function SpendMetrics() {
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
        response: costTrend,
        isLoading: costTrendLoading,
        error: costTrendError,
        sendNow: costTrendRefresh,
    } = useInventoryApiV2AnalyticsSpendTrendList({
        ...query,
        granularity: chartGranularity,
    })

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

    const { response, isLoading } = useInventoryApiV2AnalyticsSpendTableList({
        startTime: activeTimeRange.start.unix(),
        endTime: activeTimeRange.end.unix(),
        dimension: 'metric',
        granularity: tableGranularity,
        connector: [selectedConnections.provider],
        connectionId: selectedConnections.connections,
        connectionGroup: selectedConnections.connectionGroup,
    })

    const { response: responsePrev, isLoading: isLoadingPrev } =
        useInventoryApiV2AnalyticsSpendTableList({
            startTime: prevTimeRange.start.unix(),
            endTime: prevTimeRange.end.unix(),
            dimension: 'metric',
            granularity: tableGranularity,
            connector: [selectedConnections.provider],
            connectionId: selectedConnections.connections,
            connectionGroup: selectedConnections.connectionGroup,
        })

    const [chartLayout, setChartLayout] = useURLParam<ChartLayout>(
        'show',
        'metrics'
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
                        costTrend={costTrend || []}
                        title="Total spend"
                        timeRange={activeTimeRange}
                        timeRangePrev={prevTimeRange}
                        total={serviceCostResponse?.total_cost || 0}
                        totalPrev={servicePrevCostResponse?.total_cost || 0}
                        chartLayout={chartLayout}
                        setChartLayout={setChartLayout}
                        validChartLayouts={['total', 'metrics']}
                        isLoading={
                            costTrendLoading ||
                            serviceCostLoading ||
                            servicePrevCostLoading
                        }
                        error={toErrorMessage(
                            costTrendError,
                            serviceCostErr,
                            servicePrevCostErr
                        )}
                        onRefresh={() => {
                            costTrendRefresh()
                            serviceCostPrevRefresh()
                            serviceCostRefresh()
                        }}
                        onGranularityChanged={setChartGranularity}
                    />
                </Col>
                <Col numColSpan={3} className="mt-6">
                    <MetricTable
                        timeRange={activeTimeRange}
                        prevTimeRange={prevTimeRange}
                        isLoading={isLoading || isLoadingPrev}
                        response={response}
                        responsePrev={responsePrev}
                        onGranularityChange={setTableGranularity}
                        selectedGranularity={tableGranularity}
                    />
                </Col>
            </Grid>
        </>
    )
}
