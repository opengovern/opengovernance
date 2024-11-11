import { useState } from 'react'
import { Grid } from '@tremor/react'
import { useParams } from 'react-router-dom'
import {
    ChartLayout,
    Granularity,
} from '../../../components/Spend/Chart/Selectors'
import TopHeader from '../../../components/Layout/Header'
import { toErrorMessage } from '../../../types/apierror'
import { AssetChart } from '../../../components/Asset/Chart'
import { accountTrend } from '../Overview'
import {
    useInventoryApiV2AnalyticsMetricList,
    useInventoryApiV2AnalyticsTableList,
    useInventoryApiV2AnalyticsTrendList,
} from '../../../api/inventory.gen'
import AccountTable from './Table'
import {
    defaultTime,
    useFilterState,
    useURLParam,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export default function AssetAccounts() {
    const { ws } = useParams()
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultTime(ws || '')
    )
    const { value: selectedConnections } = useFilterState()

    const [granularity, setGranularity] = useState<Granularity>('daily')
    const [chartLayout, setChartLayout] = useURLParam<ChartLayout>(
        'show',
        'accounts'
    )

    const query: {
        pageSize: number
        pageNumber: number
        sortBy: 'count' | undefined
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
        sortBy: 'count',
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
        response: trendResponse,
        isLoading: trendLoading,
        error: trendError,
        sendNow: trendRefresh,
    } = useInventoryApiV2AnalyticsTrendList({
        ...query,
        granularity,
    })

    const {
        response: serviceResponse,
        isLoading: serviceLoading,
        error: serviceErr,
        sendNow: serviceRefresh,
    } = useInventoryApiV2AnalyticsMetricList(query)

    const {
        response: servicePrevResponse,
        isLoading: servicePrevLoading,
        error: servicePrevErr,
        sendNow: servicePrevRefresh,
    } = useInventoryApiV2AnalyticsMetricList(prevQuery)

    const { response: responseChart, isLoading: isLoadingChart } =
        useInventoryApiV2AnalyticsTableList({
            startTime: activeTimeRange.start.unix(),
            endTime: activeTimeRange.end.unix(),
            dimension: 'connection',
            granularity,
        })

    const trend = () => {
        if (chartLayout === 'total') {
            return trendResponse || []
        }
        if (chartLayout === 'accounts' || chartLayout === 'provider') {
            return accountTrend(responseChart || [], chartLayout) || []
        }
        return []
    }
    return (
        <>
            <TopHeader
                supportedFilters={['Date', 'Cloud Account', 'Connector']}
                initialFilters={['Date']}
            />
            <Grid className="w-full gap-10">
                <AssetChart
                    trend={trend()}
                    title="Total resources"
                    timeRange={activeTimeRange}
                    timeRangePrev={prevTimeRange}
                    total={serviceResponse?.total_count || 0}
                    totalPrev={servicePrevResponse?.total_count || 0}
                    chartLayout={chartLayout}
                    setChartLayout={setChartLayout}
                    validChartLayouts={['total', 'provider', 'accounts']}
                    isLoading={
                        trendLoading || serviceLoading || servicePrevLoading
                    }
                    error={toErrorMessage(
                        trendError,
                        serviceErr,
                        servicePrevErr
                    )}
                    onRefresh={() => {
                        trendRefresh()
                        servicePrevRefresh()
                        serviceRefresh()
                    }}
                    onGranularityChanged={setGranularity}
                />
                <AccountTable
                    timeRange={activeTimeRange}
                    connections={selectedConnections}
                />
            </Grid>
        </>
    )
}
