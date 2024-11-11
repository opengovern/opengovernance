import { Flex, Tab, TabGroup, TabList } from '@tremor/react'
import dayjs from 'dayjs'
import Selector from '../../Selector'
import { checkGranularity } from '../../../utilities/dateComparator'
import { BarChartIcon, LineChartIcon } from '../../../icons/icons'

export type ChartType = 'bar' | 'line'
const chartTypeValues: ChartType[] = ['bar', 'line']

export type Granularity = 'daily' | 'monthly'
export type ChartLayout =
    | 'total'
    | 'categories'
    | 'metrics'
    | 'provider'
    | 'accounts'

export type Aggregation = 'trending' | 'aggregated'
const aggregationValues: Aggregation[] = ['trending', 'aggregated']

interface ISpendChartSelectors {
    timeRange: {
        start: dayjs.Dayjs
        end: dayjs.Dayjs
    }
    chartType: ChartType
    setChartType: (v: ChartType) => void
    granularity: Granularity
    setGranularity: (v: Granularity) => void
    chartLayout: ChartLayout
    setChartLayout: (v: ChartLayout) => void
    validChartLayouts: ChartLayout[]
    aggregation: Aggregation
    setAggregation: (v: Aggregation) => void
    noStackedChart?: boolean
}

export function SpendChartSelectors({
    timeRange,
    chartType,
    setChartType,
    granularity,
    setGranularity,
    chartLayout,
    setChartLayout,
    validChartLayouts,
    aggregation,
    setAggregation,
    noStackedChart,
}: ISpendChartSelectors) {
    const generateGranularityList = (
        s = timeRange.start,
        e = timeRange.end
    ) => {
        let List: string[] = []
        if (checkGranularity(s, e).daily) {
            List = [...List, 'daily']
        }
        if (checkGranularity(s, e).monthly) {
            List = [...List, 'monthly']
        }
        if (checkGranularity(s, e).yearly) {
            List = [...List, 'yearly']
        }
        return List
    }

    return (
        <Flex justifyContent="end" className="gap-0">
            <Selector
                values={generateGranularityList(timeRange.start, timeRange.end)}
                value={granularity}
                title="Granularity  "
                onValueChange={(v) => {
                    setGranularity(v as Granularity)
                }}
            />

            {noStackedChart ? null : (
                <Selector
                    values={validChartLayouts.map((v) => String(v))}
                    value={chartLayout}
                    title="Show"
                    onValueChange={(v) => {
                        setChartLayout(v as ChartLayout)
                    }}
                />
            )}

            <Selector
                values={aggregationValues.map((v) => String(v))}
                value={aggregation}
                title="View"
                onValueChange={(v) => {
                    setAggregation(v as Aggregation)
                }}
            />

            <TabGroup
                index={chartTypeValues.indexOf(chartType)}
                onIndexChange={(i) =>
                    setChartType(chartTypeValues.at(i) || 'bar')
                }
                className="w-fit rounded-lg ml-2"
            >
                <TabList variant="solid">
                    <Tab value="bar">
                        <BarChartIcon className="h-4 w-4 m-0.5 my-1.5" />
                    </Tab>
                    <Tab value="line">
                        <LineChartIcon className="h-4 w-4 m-0.5 my-1.5" />
                    </Tab>
                </TabList>
            </TabGroup>
        </Flex>
    )
}
