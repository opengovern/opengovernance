// @ts-nocheck
import {
    Flex,
    Text,
    Title,
    ProgressCircle,
    Button,
    Grid,
    Subtitle,
    Card,
    Icon,
} from '@tremor/react'
import { useSetAtom } from 'jotai'
import { useEffect, useState } from 'react'
import { useScheduleApiV1ComplianceTriggerUpdate } from '../../../api/schedule.gen'
import { notificationAtom } from '../../../store'
import ScoreCategoryCard from '../../../components/Cards/ScoreCategoryCard'
import { useComplianceApiV1BenchmarksSummaryList } from '../../../api/compliance.gen'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary,
    GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkStatusResult,
    SourceType,
} from '../../../api/api'
import { useFilterState } from '../../../utilities/urlstate'
import { getErrorMessage, toErrorMessage } from '../../../types/apierror'
import KPICard from './KPICard'
import { useParams } from 'react-router-dom'
import { ChevronDoubleUpIcon } from '@heroicons/react/24/outline'
import axios from 'axios'

function SecurityScore(
    v:
        | GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkStatusResult[]
        | undefined
) {
    const total =
        v?.map((t) => t.total || 0).reduce((prev, curr) => prev + curr, 0) || 0
    const passed =
        v?.map((t) => t.passed || 0).reduce((prev, curr) => prev + curr, 0) || 0

    if (total === 0) {
        return 0
    }
    return (passed / total) * 100
}

function fixSort(t: string) {
    return t
        .replaceAll('s', 'a')
        .replaceAll('S', 'a')
        .replaceAll('c', 'b')
        .replaceAll('C', 'b')
        .replaceAll('o', 'c')
        .replaceAll('O', 'c')
        .replaceAll('r', 'd')
        .replaceAll('R', 'd')
        .replaceAll('E', 'e')
}

interface MR {
    category: string
    title: string
    summary: GithubComKaytuIoKaytuEnginePkgComplianceApiBenchmarkEvaluationSummary[]
}

export default function ScoreKPIs() {
    const { value: selectedConnections } = useFilterState()
    const { ws } = useParams()

    const setNotification = useSetAtom(notificationAtom)
  const [response, setResponse] = useState()
  const [isLoading, setIsLoading] = useState(false)
    


  const GetBenchmarks = (benchmarks: string[]) => {
      setIsLoading(true)
      let url = ''
      if (window.location.origin === 'http://localhost:3000') {
          url = window.__RUNTIME_CONFIG__.REACT_APP_BASE_URL
      } else {
          url = window.location.origin
      }
      // @ts-ignore
      const token = JSON.parse(localStorage.getItem('openg_auth')).token

      const config = {
          headers: {
              Authorization: `Bearer ${token}`,
          },
      }
      const body = {
          benchmarks: benchmarks,
      }
      axios
          .post(
              `${url}/main/compliance/api/v3/compliance/summary/benchmark`,
              body,
              config
          )
          .then((res) => {
              //  const temp = []
              setIsLoading(false)
              setResponse(res.data)
          })
          .catch((err) => {
              setIsLoading(false)

              console.log(err)
          })
  }
     useEffect(() => {
         GetBenchmarks([
             'sre_efficiency',
             'sre_reliability',
             'sre_supportability',
         ])
     }, [])

   

    return (
        <>
            <Card>
                <div className="flex items-center justify-start">
                    <Icon icon={ChevronDoubleUpIcon} className="w-6 h-6" />
                    <Title className="font-semibold">SRE</Title>
                </div>
                <div
                    className={'h-fit'}
                    style={{
                        transitionDuration: '300ms',
                        animationFillMode: 'backwards',
                    }}
                >
                    <Grid
                        numItems={3}
                        className="mt-6 grid grid-cols-1 gap-4 rounded-md bg-gray-50 py-4 dark:bg-gray-900 md:grid-cols-3 md:divide-x md:divide-gray-200 md:dark:divide-gray-800"
                    >
                        {isLoading || !response ? (
                            <>
                                {/* <Flex
                            flexDirection="col"
                            className="border px-8 py-6 rounded-lg"
                            alignItems="start"
                        >
                            <Text className="text-gray-500">
                                <span className="font-bold text-gray-800 mr-1.5">
                                    SCORE
                                </span>
                                evaluates cloud environments for alignment with
                                internal policies, vendor recommendations, and
                                industry standards
                            </Text>
                        </Flex> */}

                                {[1, 2, 3].map((i) => (
                                    <Flex
                                        alignItems="start"
                                        justifyContent="start"
                                        className="pl-5 pr-8 py-6 rounded-lg bg-white gap-5 shadow-sm hover:shadow-md"
                                    >
                                        <Flex className="relative w-fit">
                                            <ProgressCircle value={0} size="md">
                                                <div className="animate-pulse h-8 w-8 my-2 bg-slate-200 dark:bg-slate-700 rounded-full" />
                                            </ProgressCircle>
                                        </Flex>

                                        <Flex
                                            alignItems="start"
                                            flexDirection="col"
                                            className="gap-1.5"
                                        >
                                            <div className="animate-pulse h-3 w-full my-2 bg-slate-200 dark:bg-slate-700 rounded" />
                                            <div className="animate-pulse h-3 w-full my-2 bg-slate-200 dark:bg-slate-700 rounded" />
                                        </Flex>
                                    </Flex>
                                ))}
                            </>
                        ) : (
                            <>
                                {/* <Flex
                            flexDirection="col"
                            className="border px-8 py-6 rounded-lg"
                            alignItems="start"
                        >
                            <Text className="text-gray-500">
                                <span className="font-bold text-gray-800 dark:text-gray-100 mr-1.5">
                                    SCORE
                                </span>
                                evaluates cloud environments for alignment with
                                internal policies, vendor recommendations, and
                                industry standards
                            </Text>
                        </Flex> */}

                                {response
                                    .sort((a, b) => {
                                        if (
                                            a.benchmark_title ===
                                                'SRE Supportability' &&
                                            b.benchmark_title ===
                                                'SRE Efficiency'
                                        ) {
                                            return -1
                                        }
                                        if (
                                            a.benchmark_title ===
                                                'SRE Efficiency' &&
                                            b.benchmark_title ===
                                                'SRE Supportability'
                                        ) {
                                            return 1
                                        }
                                        if (
                                            a.benchmark_title ===
                                                'SRE Reliability' &&
                                            b.benchmark_title ===
                                                'SRE Efficiency'
                                        ) {
                                            return -1
                                        }
                                        if (
                                            a.benchmark_title ===
                                                'SRE Efficiency' &&
                                            b.benchmark_title ===
                                                'SRE Reliability'
                                        ) {
                                            return 1
                                        }
                                        if (
                                            a.benchmark_title ===
                                                'SRE Supportability' &&
                                            b.benchmark_title ===
                                                'SRE Reliability'
                                        ) {
                                            return -1
                                        }
                                        return 0
                                    })
                                    .map((item) => {
                                        return (
                                            <KPICard
                                                link={`/compliance/${item.benchmark_id}`}
                                                name={item.benchmark_title
                                                    .split('SRE')[1]
                                                    .trim()}
                                                number={
                                                    item.benchmark_id ===
                                                    'sre_efficiency'
                                                        ? item.cost_optimization
                                                        : item.issues_count
                                                }
                                                percentage={
                                                    (item
                                                        .severity_summary_by_control
                                                        .total.passed /
                                                        item
                                                            .severity_summary_by_control
                                                            .total.total) *
                                                    100
                                                }
                                            />
                                        )
                                    })}
                            </>
                        )}
                    </Grid>
                </div>
            </Card>

            
        </>
    )
}

{
    /* <ScoreCategoryCard
                                    title={item.title || ''}
                                    percentage={SecurityScore(
                                        item.summary.map(
                                            (c) =>
                                                c.controlsSeverityStatus
                                                    ?.total || {}
                                        )
                                    )}
                                    costOptimization={
                                        item.category === 'cost_optimization'
                                            ? item.summary
                                                  .map(
                                                      (b) =>
                                                          b.costOptimization ||
                                                          0
                                                  )
                                                  .reduce<number>(
                                                      (prev, curr) =>
                                                          prev + curr,
                                                      0
                                                  )
                                            : 0
                                    }
                                    value={item.summary
                                        .map(
                                            (c) =>
                                                c.controlsSeverityStatus?.total
                                                    ?.passed || 0
                                        )
                                        .reduce<number>(
                                            (prev, curr) => prev + curr,
                                            0
                                        )}
                                    kpiText="Issues"
                                    varient="minimized"
                                    category={item.category}
                                /> */
}
