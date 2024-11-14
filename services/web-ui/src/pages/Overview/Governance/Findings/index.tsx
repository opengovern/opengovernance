import { Button, Divider, Flex, Text, Title } from '@tremor/react'
import { ChevronRightIcon } from '@heroicons/react/20/solid'
import { useNavigate, useParams } from 'react-router-dom'
import { useAtomValue } from 'jotai'
import { useComplianceApiV1ControlsSummaryList } from '../../../../api/compliance.gen'
import { TypesFindingSeverity } from '../../../../api/api'
import { severityBadge } from '../../../Governance/Controls'
import { getErrorMessage } from '../../../../types/apierror'
import { searchAtom } from '../../../../utilities/urlstate'

export default function Findings() {
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const {
        response,
        isLoading,
        error,
        sendNow: refresh,
    } = useComplianceApiV1ControlsSummaryList()

    const critical = Array.isArray(response)
        ? response?.filter(
              (item) =>
                  item.control?.severity ===
                      TypesFindingSeverity.FindingSeverityCritical &&
                  item.passed === false
          ) || []
        : []

    const high = Array.isArray(response)
        ? response?.filter(
              (item) =>
                  item.control?.severity ===
                      TypesFindingSeverity.FindingSeverityHigh &&
                  item.passed === false
          ) || []
        : []

    const medium = Array.isArray(response)
        ? response?.filter(
              (item) =>
                  item.control?.severity ===
                      TypesFindingSeverity.FindingSeverityMedium &&
                  item.passed === false
          ) || []
        : []

    const controls = critical.concat(high).concat(medium).slice(0, 3)

    return (
        <>
            <Flex>
                <Title className="text-gray-500">Problematic Controls</Title>
                <Button
                    variant="light"
                    icon={ChevronRightIcon}
                    iconPosition="right"
                    onClick={() =>
                        navigate(`/incidents?${searchParams}`)
                    }
                >
                    Show all
                </Button>
            </Flex>
            <Flex
                flexDirection="col"
                className={`mt-4 ${isLoading ? 'animate-pulse' : ''}`}
            >
                {isLoading || getErrorMessage(error).length > 0
                    ? [1, 2, 3].map((i, idx, arr) => {
                          return (
                              <>
                                  <Flex
                                      flexDirection="col"
                                      justifyContent="start"
                                      alignItems="start"
                                      className="w-full py-4 px-4"
                                  >
                                      <div className="h-2 w-72 my-1 bg-slate-200 dark:bg-slate-700 rounded" />
                                      <Flex flexDirection="row">
                                          <div className="h-6 w-16 my-1 bg-slate-200 dark:bg-slate-700 rounded-md" />
                                          <div className="h-6 w-36 my-1 bg-slate-200 dark:bg-slate-700 rounded-md" />
                                      </Flex>
                                  </Flex>
                                  {idx + 1 < arr.length && (
                                      <Divider className="m-0 p-0" />
                                  )}
                              </>
                          )
                      })
                    : controls.map((item, idx, arr) => {
                          return (
                              <>
                                  <Flex
                                      className="py-4 px-4 hover:bg-gray-100 dark:hover:bg-gray-900 rounded-md cursor-pointer"
                                      onClick={() =>
                                          navigate(
                                              `/incidents/${item.control?.id}?${searchParams}`
                                          )
                                      }
                                  >
                                      <Flex
                                          flexDirection="col"
                                          alignItems="start"
                                      >
                                          <Text className="w-3/4 line-clamp-1 text-black mb-2">
                                              {item.control?.title}
                                          </Text>
                                          <Text>
                                              # of failed resources:{' '}
                                              {item.failedResourcesCount}
                                          </Text>
                                      </Flex>
                                      {severityBadge(item.control?.severity)}
                                  </Flex>
                                  {idx + 1 < arr.length && (
                                      <Divider className="m-0 mb-0 p-0" />
                                  )}
                              </>
                          )
                      })}
            </Flex>
            {error && (
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
                        <Text className="mt-2">{getErrorMessage(error)}</Text>
                    </Flex>
                    <Button
                        variant="secondary"
                        className="mb-6"
                        color="slate"
                        onClick={refresh}
                    >
                        Try Again
                    </Button>
                </Flex>
            )}
        </>
    )
}
