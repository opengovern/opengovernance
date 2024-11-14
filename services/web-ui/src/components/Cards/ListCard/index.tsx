import { ChevronRightIcon } from '@heroicons/react/24/outline'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
    Button,
    Card,
    Flex,
    List,
    ListItem,
    Tab,
    TabGroup,
    TabList,
    TabPanel,
    TabPanels,
    Text,
    Title,
} from '@tremor/react'
import { useAtomValue } from 'jotai'
import { SourceType } from '../../../api/api'
import { numericDisplay } from '../../../utilities/numericDisplay'
import { getConnectorsIcon } from '../ConnectorCard'
import { isDemoAtom } from '../../../store'
import { errorHandlingWithErrorMessage } from '../../../types/apierror'
import { searchAtom } from '../../../utilities/urlstate'
import BadgeDeltaSimple from '../../ChangeDeltaSimple'

interface ITopListCard {
    title: string
    showColumnsTitle: boolean
    keyColumnTitle?: string
    valueColumnTitle?: string
    changeColumnTitle?: string
    tabs?: string[]
    onTabChange?: (tabIdx: number) => void
    loading: boolean
    isPercentage?: boolean
    isPrice?: boolean
    items: {
        data: {
            name: string | undefined
            value: number | undefined
            valueRateChange?: number | undefined
            connector?: SourceType[]
            id?: string | undefined
        }[]
        total: number | undefined
    }
    url?: string
    type: 'service' | 'account'
    isClickable?: boolean
    linkPrefix?: string
    error?: string | undefined
    onRefresh?: () => void
}

interface Item {
    name: string | undefined
    value: number | undefined
    valueRateChange?: number | undefined
    connector?: SourceType[]
    id?: string | undefined
    kaytuId?: string | undefined
}

export default function ListCard({
    title,
    showColumnsTitle,
    keyColumnTitle,
    valueColumnTitle,
    changeColumnTitle,
    tabs,
    onTabChange,
    loading,
    isPrice,
    isPercentage,
    items,
    url,
    type,
    isClickable = true,
    linkPrefix,
    error,
    onRefresh,
}: ITopListCard) {
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const isDemo = useAtomValue(isDemoAtom)

    const value = (item: Item) => {
        if (isPercentage) {
            return item.value
        }
        if (isPrice) {
            return `$${numericDisplay(item.value)}`
        }
        return numericDisplay(item.value)
    }

    return (
        <Card className={`h-full `}>
            <Flex flexDirection="col" alignItems="start" className="h-full">
                <Flex flexDirection="col" alignItems="start">
                    <TabGroup onIndexChange={onTabChange}>
                        <Flex alignItems="start" className="mb-6">
                            <Title className="font-semibold">{title}</Title>
                            {tabs && tabs.length > 0 && (
                                <TabList variant="solid">
                                    {tabs.map((v) => (
                                        <Tab>{v}</Tab>
                                    ))}
                                </TabList>
                            )}
                        </Flex>
                    </TabGroup>
                    <Flex flexDirection="col" justifyContent="start">
                        {showColumnsTitle && (
                            <Flex
                                alignItems="baseline"
                                justifyContent="between"
                                className="space-x-0 mb-2 pr-1"
                            >
                                <Text className="font-medium px-1 text-gray-400 dark:text-gray-500">
                                    {keyColumnTitle}
                                </Text>
                                <Flex className=" w-fit">
                                    <Text className="w-20 text-right font-medium text-gray-400 dark:text-gray-500">
                                        {valueColumnTitle}
                                    </Text>
                                    {changeColumnTitle && (
                                        <Text className="w-20 text-right font-medium text-gray-400 dark:text-gray-500">
                                            {changeColumnTitle}
                                        </Text>
                                    )}
                                </Flex>
                            </Flex>
                        )}

                        {loading ? (
                            <List className="animate-pulse">
                                {[1, 2, 3, 4, 5].map((i) => (
                                    <ListItem className="max-w-full p-1 py-3">
                                        <Flex
                                            flexDirection="row"
                                            justifyContent="between"
                                            className="py-1.5"
                                        >
                                            <div className="h-2 w-52 my-1 bg-slate-200 dark:bg-slate-700 rounded" />
                                            <div className="h-2 w-16 my-1 bg-slate-200 dark:bg-slate-700 rounded" />
                                        </Flex>
                                    </ListItem>
                                ))}
                            </List>
                        ) : (
                            <List>
                                {items?.data.map((item: Item) => (
                                    <ListItem
                                        key={item.name}
                                        className={`max-w-full p-1 ${
                                            isClickable
                                                ? 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-900'
                                                : ''
                                        } ${
                                            (item.connector?.length || 0) > 0
                                                ? ''
                                                : 'py-3'
                                        }`}
                                        onClick={() =>
                                            isClickable
                                                ? navigate(
                                                      `${
                                                          linkPrefix !==
                                                          undefined
                                                              ? linkPrefix
                                                              : ''
                                                      }${
                                                          type === 'account'
                                                              ? 'account_'
                                                              : 'metric_'
                                                      }${
                                                          item.kaytuId
                                                      }?${searchParams}`
                                                  )
                                                : undefined
                                        }
                                    >
                                        <Flex
                                            className="py-1"
                                            justifyContent="between"
                                        >
                                            <Flex
                                                justifyContent="start"
                                                className="w-fit"
                                            >
                                                {getConnectorsIcon(
                                                    item.connector || []
                                                )}
                                                <Text
                                                    className={
                                                        type === 'account' &&
                                                        isDemo
                                                            ? 'text-gray-800 ml-2 truncate blur-sm'
                                                            : 'text-gray-800 ml-2 truncate'
                                                    }
                                                >
                                                    {item.name}
                                                </Text>
                                            </Flex>
                                            <Flex className="w-fit">
                                                {item.value && (
                                                    <Text className="text-gray-800 w-20 text-right min-w-fit">
                                                        {value(item)}
                                                    </Text>
                                                )}
                                                {item.valueRateChange !==
                                                    undefined && (
                                                    <Flex
                                                        className="w-20"
                                                        justifyContent="end"
                                                    >
                                                        <BadgeDeltaSimple
                                                            change={
                                                                item.valueRateChange
                                                            }
                                                            maxChange={999}
                                                        />
                                                    </Flex>
                                                )}
                                            </Flex>
                                        </Flex>
                                    </ListItem>
                                ))}
                            </List>
                        )}
                    </Flex>
                </Flex>
                <Flex
                    justifyContent="end"
                    className="cursor-pointer mt-2"
                    onClick={() => {
                        if (url) {
                            navigate(
                                url.indexOf('?') > 0
                                    ? `${url}&${searchParams}`
                                    : `${url}?${searchParams}`
                            )
                        }
                    }}
                >
                    {(items.total || 0) - items.data.length > 0 && (
                        <Button
                            variant="secondary"
                            icon={ChevronRightIcon}
                            iconPosition="right"
                        >
                            {`+ ${numericDisplay(
                                (items.total || 0) - items.data.length
                            )} more`}
                        </Button>
                    )}
                </Flex>
            </Flex>
            {errorHandlingWithErrorMessage(onRefresh, error)}
        </Card>
    )
}
