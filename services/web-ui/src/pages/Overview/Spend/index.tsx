import {
    Badge,
    BadgeDelta,
    Button,
    Card,
    Col,
    Flex,
    Grid,
    Icon,
    Metric,
    Subtitle,
    Text,
    Title,
} from '@tremor/react'
import { BanknotesIcon } from '@heroicons/react/24/outline'
import { ChevronRightIcon } from '@heroicons/react/20/solid'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useAtomValue } from 'jotai'
import {
    useInventoryApiV2AnalyticsSpendMetricList,
    useInventoryApiV2AnalyticsSpendTrendList,
} from '../../../api/inventory.gen'
import { getErrorMessage, toErrorMessage } from '../../../types/apierror'
import { buildTrend } from '../../../components/Spend/Chart/helpers'
import StackedChart from '../../../components/Chart/Stacked'
import { exactPriceDisplay } from '../../../utilities/numericDisplay'
import { renderDateText } from '../../../components/Layout/Header/DatePicker'
import ChangeDelta from '../../../components/ChangeDelta'
import {
    defaultHomepageTime,
    defaultSpendTime,
    searchAtom,
    useFilterState,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export default function Spend() {
   
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultHomepageTime()
    )
    const { value: selectedConnections } = useFilterState()
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

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
        granularity: 'daily',
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
    const trendStacked = Array.isArray(costTrend)
        ? buildTrend(costTrend || [], 'trending', 'daily', 4)
        : {
              label: [],
              data: [],
              flag: [],
          }

    return (
        <Card className=" relative border-0 ring-0 !shadow-sm">
            <Flex flexDirection="col" className="gap-2 px-2">
                {/* <Flex justifyContent="start" className="gap-2">
                            <Icon icon={BanknotesIcon} className="p-0" />
                            <Title className="font-semibold">Cloud Spend</Title>
                        </Flex> */}
                <Flex>
                    <Title className="text-gray-500">Cloud Spend</Title>
                    <Button
                        variant="light"
                        icon={ChevronRightIcon}
                        iconPosition="right"
                        onClick={() =>
                            navigate(`/spend?${searchParams}`)
                        }
                    >
                        View details
                    </Button>
                </Flex>

                {serviceCostLoading ? (
                    <Flex
                        justifyContent="start"
                        alignItems="baseline"
                        className="animate-pulse gap-4 mb-6"
                    >
                        <div className="h-8 w-36 bg-slate-200 dark:bg-slate-700 rounded" />
                        <div className="h-4 w-20 bg-slate-200 dark:bg-slate-700 rounded" />
                    </Flex>
                ) : (
                    <Flex
                        justifyContent="start"
                        alignItems="baseline"
                        className="gap-4 mb-6"
                    >
                        <Metric>
                            {exactPriceDisplay(
                                serviceCostResponse?.total_cost || 0,
                                0
                            )}
                        </Metric>

                        <ChangeDelta
                            change={
                                (((serviceCostResponse?.total_cost || 0) -
                                    (servicePrevCostResponse?.total_cost ||
                                        0)) /
                                    (servicePrevCostResponse?.total_cost ||
                                        1)) *
                                100
                            }
                            size="sm"
                            valueInsideBadge
                        />
                        {/* <Text className="text-xs mt-2 text-gray-400">
                                {`Compared to ${renderDateText(
                                    prevTimeRange.start,
                                    prevTimeRange.end
                                )}`}
                            </Text> */}
                    </Flex>
                )}

                <StackedChart
                    labels={trendStacked.label}
                    chartData={trendStacked.data}
                    isCost
                    chartType="line"
                    loading={
                        costTrendLoading ||
                        serviceCostLoading ||
                        servicePrevCostLoading
                    }
                    error={toErrorMessage(
                        costTrendError,
                        serviceCostErr,
                        servicePrevCostErr
                    )}
                />
            </Flex>
            {/*            <Flex justifyContent="start" className="gap-4 w-full flex-wrap">
                {trendStacked.data[0] ? (
                    trendStacked.data[0].map((t, i) => (
                        <div>
                            <Flex
                                justifyContent="start"
                                className="gap-2 w-fit"
                            >
                                <div
                                    className="h-2 w-2 min-w-[8px] rounded-full"
                                    style={{
                                        backgroundColor: colors[i],
                                    }}
                                />
                                <Text className="truncate">{t.label}</Text>
                            </Flex>
                        </div>
                    ))
                ) : (
                    <div className="h-6" />
                )}
            </Flex> */}
            {toErrorMessage(
                costTrendError,
                serviceCostErr,
                servicePrevCostErr
            ) && (
                <Flex
                    flexDirection="col"
                    justifyContent="between"
                    className="absolute top-0 w-full left-0 h-full backdrop-blur"
                >
                    <Flex
                        flexDirection="col"
                        justifyContent="center"
                        alignItems="center"
                    >
                        <Title className="mt-6">Failed to load component</Title>
                        <Text className="mt-2">
                            {toErrorMessage(
                                costTrendError,
                                serviceCostErr,
                                servicePrevCostErr
                            )}
                        </Text>
                    </Flex>
                    <Button
                        variant="secondary"
                        className="mb-6"
                        color="slate"
                        onClick={() => {
                            serviceCostRefresh()
                            serviceCostPrevRefresh()
                            costTrendRefresh()
                        }}
                    >
                        Try Again
                    </Button>
                </Flex>
            )}
        </Card>
    )
}
