import { Card, Flex, Tab, TabGroup, TabList, Title } from '@tremor/react'
import { trendChart } from './helpers'
import StackedChart from '../../Chart/Stacked'
import Selector from '../../Selector'
import { ChartType, chartTypeValues } from '../../Asset/Chart/Selectors'
import { BarChartIcon, LineChartIcon } from '../../../icons/icons'
import { errorHandlingWithErrorMessage } from '../../../types/apierror'
import { useURLParam } from '../../../utilities/urlstate'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkStatusResult,
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkTrendDatapoint,
    GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatusSummary,
} from '../../../api/api'
import { camelCaseToLabel } from '../../../utilities/labelMaker'

export type BenchmarkChartShowType =
    | 'Conformance Status'
    | 'Severity'
    | 'Security Score'
export const BenchmarkChartShowValues = [
    'Conformance Status',
    'Severity',
    'Security Score',
]

export type BenchmarkChartViewType = 'Findings' | 'Controls'
export const BenchmarkChartViewValues = ['Findings', 'Controls']

export type BenchmarkChartIncludePassedType = 'True' | 'False'
export const BenchmarkChartIncludePassedValues = ['True', 'False']

export interface ITrendItem {
    stack: IStackItem[]
    timestamp: string | undefined
}

export interface IStackItem {
    name: string
    count: number
}

const failed = (
    v:
        | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkStatusResult
        | undefined
) => {
    return (v?.total || 0) - (v?.passed || 0)
}

const score = (
    v:
        | GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatusSummary
        | undefined
) => {
    return (100.0 * (v?.passed || 0)) / ((v?.failed || 0) + (v?.passed || 0))
}

const benchmarkTrend = (
    response:
        | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkTrendDatapoint[]
        | undefined,
    includePassed: 'True' | 'False',
    show: 'Conformance Status' | 'Severity' | 'Security Score',
    view: 'Findings' | 'Controls'
): ITrendItem[] => {
    return (
        response?.map((item) => {
            if (show === 'Security Score') {
                const stack: IStackItem[] = [
                    {
                        name: 'Score',
                        count: score(item.conformanceStatusSummary),
                    },
                ]
                return {
                    timestamp: item.timestamp,
                    stack,
                }
            }

            if (view === 'Findings') {
                if (show === 'Conformance Status') {
                    const data = {
                        failed: item.conformanceStatusSummary?.failed,
                        passed: item.conformanceStatusSummary?.passed,
                    }

                    const stack: IStackItem[] = Object.entries(data).map(
                        ([key, value]) => {
                            return {
                                name: camelCaseToLabel(key),
                                count: value || 0,
                            }
                        }
                    )
                    return {
                        timestamp: item.timestamp,
                        stack,
                    }
                }

                const data = {
                    critical: item.checks?.criticalCount,
                    high: item.checks?.highCount,
                    medium: item.checks?.mediumCount,
                    low: item.checks?.lowCount,
                    none: item.checks?.noneCount,
                }

                const stack: IStackItem[] = Object.entries(data).map(
                    ([key, value]) => {
                        return {
                            name: camelCaseToLabel(key),
                            count: value || 0,
                        }
                    }
                )

                if (includePassed === 'True') {
                    stack.push({
                        name: 'Passed',
                        count: item.conformanceStatusSummary?.passed || 0,
                    })
                }
                return {
                    timestamp: item.timestamp,
                    stack,
                }
            }

            if (view === 'Controls') {
                if (show === 'Conformance Status') {
                    const data = {
                        failed: failed(item.controlsSeverityStatus?.total),
                        passed: item.controlsSeverityStatus?.total?.passed,
                    }

                    const stack: IStackItem[] = Object.entries(data).map(
                        ([key, value]) => {
                            return {
                                name: camelCaseToLabel(key),
                                count: value || 0,
                            }
                        }
                    )
                    return {
                        timestamp: item.timestamp,
                        stack,
                    }
                }

                const data = {
                    critical: failed(item.controlsSeverityStatus?.critical),
                    high: failed(item.controlsSeverityStatus?.high),
                    medium: failed(item.controlsSeverityStatus?.medium),
                    low: failed(item.controlsSeverityStatus?.low),
                    none: failed(item.controlsSeverityStatus?.none),
                }

                const stack: IStackItem[] = Object.entries(data).map(
                    ([key, value]) => {
                        return {
                            name: camelCaseToLabel(key),
                            count: value || 0,
                        }
                    }
                )

                if (includePassed === 'True') {
                    stack.push({
                        name: 'Passed',
                        count: item.controlsSeverityStatus?.total?.passed || 0,
                    })
                }
                return {
                    timestamp: item.timestamp,
                    stack,
                }
            }

            return {
                timestamp: item.timestamp,
                stack: [],
            }
        }) || []
    )
}

interface IBenchmarkChart {
    title: string
    isLoading: boolean
    trend:
        | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkTrendDatapoint[]
        | undefined
    error: string | undefined
    onRefresh: () => void
}

export default function BenchmarkChart({
    title,
    isLoading,
    trend,
    error,
    onRefresh,
}: IBenchmarkChart) {
    const [includePassed, setIncludePassed] = useURLParam<'True' | 'False'>(
        'includePassed',
        'False'
    )
    const [show, setShow] = useURLParam<
        'Conformance Status' | 'Severity' | 'Security Score'
    >('show', 'Severity')
    const [view, setView] = useURLParam<'Findings' | 'Controls'>(
        'view',
        'Controls'
    )

    const [chartType, setChartType] = useURLParam<ChartType>(
        'chartType',
        'line'
    )

    const theTrend = trendChart(
        benchmarkTrend(trend, includePassed, show, view)
    )

    return (
        <Card className="mb-6">
            <Flex>
                <Title>{title}</Title>
                <Flex className="w-fit gap-6">
                    <Selector
                        values={[
                            'Conformance Status',
                            'Severity',
                            'Security Score',
                        ]}
                        value={show}
                        title="Show"
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        onValueChange={(v) => setShow(v)}
                    />

                    {show !== 'Security Score' && (
                        <Selector
                            values={['Findings', 'Controls']}
                            value={view}
                            title="View"
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            onValueChange={(v) => setView(v)}
                        />
                    )}

                    {show === 'Severity' && (
                        <Selector
                            values={['True', 'False']}
                            value={includePassed}
                            title="Include passed"
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            onValueChange={(v) => setIncludePassed(v)}
                        />
                    )}

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
            </Flex>
            <StackedChart
                labels={theTrend.label}
                chartData={theTrend.data}
                chartType={chartType}
                loading={isLoading}
                error={error}
                isCost={false}
                colors={theTrend.colors}
            />
            {errorHandlingWithErrorMessage(onRefresh, error)}
        </Card>
    )
}
