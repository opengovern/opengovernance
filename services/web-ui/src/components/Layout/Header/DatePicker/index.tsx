import { getLocalTimeZone, parseDate, today } from '@internationalized/date'
import { useAtomValue } from 'jotai'
import { useEffect, useRef, useState } from 'react'
import { useDateRangePickerState } from 'react-stately'
import { useDateRangePicker } from 'react-aria'
import { CalendarIcon, ChevronRightIcon } from '@heroicons/react/24/outline'
import { AriaDateRangePickerProps, DateValue } from '@react-aria/datepicker'
import dayjs from 'dayjs'
import quarterOfYear from 'dayjs/plugin/quarterOfYear'
import { Flex, Text, Title } from '@tremor/react'
import { useParams } from 'react-router-dom'
import { workspaceAtom } from '../../../../store'
import { FieldButton } from './Button'
import { RangeCalendar } from './Calendar/RangePicker/RangeCalendar'
import { Popover } from './Popover'
import { Dialog } from './Dialog'
import {
    defaultSpendTime,
    defaultTime,
    useUrlDateRangeState,
} from '../../../../utilities/urlstate'

dayjs.extend(quarterOfYear)

export const renderDateText = (st: dayjs.Dayjs, en: dayjs.Dayjs) => {
    const s = st
    const e = en
    const startYear = s.year()
    const endYear = e.year()
    const startMonth = s.month()
    const endMonth = e.month()
    const startDay = s.date()
    const endDay = e.date()

    if (startYear === endYear && startYear === dayjs().utc().year()) {
        if (startMonth === endMonth) {
            if (startDay === endDay) {
                return `${s.format('MMM')} ${startDay}`
            }
            return `${s.format('MMM')} ${startDay} - ${endDay}`
        }
        return `${s.format('MMM')} ${startDay} - ${e.format('MMM')} ${endDay}`
    }
    return `${s.format('MMM')} ${startDay}, ${startYear} - ${e.format(
        'MMM'
    )} ${endDay}, ${endYear}`
}

export function CustomDatePicker(props: AriaDateRangePickerProps<DateValue>) {
    const { ws } = useParams()
    const url = window.location.pathname.split('/')
    if (url[1] === 'ws') {
        url.shift()
    }
    const isSpend =
        url && url[2] ? url[2].includes('spend') || url[2] === 'home' : false
    const state = useDateRangePickerState(props)
    const ref = useRef(null)
    const { setValue: setActiveTimeRange } = useUrlDateRangeState(
        isSpend ? defaultSpendTime(ws || '') : defaultTime(ws || '')
    )

    const [showList, setShowList] = useState(false)
    const listState = {
        isOpen: showList,
        close: () => setShowList(false),
        setOpen: () => setShowList(true),
    }

    const { groupProps, labelProps, buttonProps, dialogProps, calendarProps } =
        useDateRangePicker(props, state, ref)

    const { label } = props
    const { value } = props
    const start = () => {
        const day = value?.start.day || 1
        const month = value?.start.month || 1
        const year = value?.start.year || 1

        return dayjs.utc(new Date(year, month - 1, day, 12)).startOf('day')
    }

    const end = () => {
        const day = value?.end.day || 1
        const month = value?.end.month || 1
        const year = value?.end.year || 1

        return dayjs.utc(new Date(year, month - 1, day, 12)).startOf('day')
    }

    const last7Days = () => {
        return {
            start: dayjs().utc().subtract(1, 'week').startOf('day'),
            end: dayjs().utc().endOf('day'),
        }
    }

    const last30Days = () => {
        return {
            start: dayjs().utc().subtract(1, 'month').startOf('day'),
            end: dayjs().utc().endOf('day'),
        }
    }

    const thisMonth = () => {
        return {
            start: dayjs().utc().startOf('month').startOf('day'),
            end: dayjs().utc().endOf('day'),
        }
    }

    const lastMonth = () => {
        return {
            start: dayjs()
                .utc()
                .subtract(1, 'month')
                .startOf('month')
                .startOf('day'),
            end: dayjs().utc().subtract(1, 'month').endOf('month').endOf('day'),
        }
    }

    const thisQuarter = () => {
        return {
            start: dayjs().utc().startOf('quarter').startOf('day'),
            end: dayjs().utc().endOf('day'),
        }
    }

    const lastQuarter = () => {
        return {
            start: dayjs()
                .utc()
                .subtract(1, 'quarter')
                .startOf('quarter')
                .startOf('day'),
            end: dayjs()
                .utc()
                .subtract(1, 'quarter')
                .endOf('quarter')
                .endOf('day'),
        }
    }

    const thisYear = () => {
        return {
            start: dayjs().utc().startOf('year').startOf('day'),
            end: dayjs().utc().endOf('day'),
        }
    }

    return (
        <div className="relative inline-flex flex-col text-left">
            <span {...labelProps} className="text-sm text-gray-800">
                {label}
            </span>
            <div
                {...groupProps}
                ref={ref}
                className="flex group h-[38px] rounded-r-lg overflow-hidden"
            >
                <div className="flex items-center bg-white dark:bg-gray-900 dark:text-gray-50 border border-gray-300 dark:border-gray-700 group-hover:border-gray-400 transition-colors rounded-l-lg px-5 group-focus-within:border-openg-600 group-focus-within:group-hover:border-openg-600 p-1 relative">
                    <Text className="text-gray-800">
                        {renderDateText(start(), end())}{' '}
                        <span className="text-orange-600 ml-2">UTC</span>
                    </Text>
                    <button
                        type="button"
                        className="absolute w-full h-full left-0 opacity-0"
                        onClick={() => listState.setOpen()}
                    >
                        open datepicker
                    </button>
                </div>
                <FieldButton {...buttonProps} isPressed={state.isOpen}>
                    <CalendarIcon className="w-5 h-5 text-gray-700 dark:text-gray-50 group-focus-within:text-openg-700" />
                </FieldButton>
            </div>
            {state.isOpen && (
                <Popover triggerRef={ref} state={state} placement="bottom end">
                    <Dialog {...dialogProps}>
                        <RangeCalendar {...calendarProps} />
                    </Dialog>
                </Popover>
            )}
            {listState.isOpen && (
                <Popover
                    triggerRef={ref}
                    state={listState}
                    placement="bottom end"
                >
                    <Flex
                        flexDirection="col"
                        alignItems="start"
                        className="gap-1 w-fit"
                    >
                        <Title>Relative dates</Title>
                        <Flex
                            onClick={() => setActiveTimeRange(last7Days())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">Last 7 days</Text>
                            <Text>
                                {renderDateText(
                                    last7Days().start,
                                    last7Days().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => setActiveTimeRange(last30Days())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">Last 30 days</Text>
                            <Text>
                                {renderDateText(
                                    last30Days().start,
                                    last30Days().end
                                )}
                            </Text>
                        </Flex>
                        <Title className="mt-3">Calender months</Title>
                        <Flex
                            onClick={() => setActiveTimeRange(thisMonth())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">This month</Text>
                            <Text>
                                {renderDateText(
                                    thisMonth().start,
                                    thisMonth().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => setActiveTimeRange(lastMonth())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">Last month</Text>
                            <Text>
                                {renderDateText(
                                    lastMonth().start,
                                    lastMonth().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => setActiveTimeRange(thisQuarter())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">This quarter</Text>
                            <Text>
                                {renderDateText(
                                    thisQuarter().start,
                                    thisQuarter().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => setActiveTimeRange(lastQuarter())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">Last quarter</Text>
                            <Text>
                                {renderDateText(
                                    lastQuarter().start,
                                    lastQuarter().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => setActiveTimeRange(thisYear())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800">This year</Text>
                            <Text>
                                {renderDateText(
                                    thisYear().start,
                                    thisYear().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            className="mt-5 space-x-4 px-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                            onClick={() => {
                                listState.close()
                                state.setOpen(true)
                            }}
                        >
                            <Text className="text-gray-800">
                                Custom date range
                            </Text>
                            <ChevronRightIcon className="h-5" />
                        </Flex>
                    </Flex>
                </Popover>
            )}
        </div>
    )
}

export default function DateRangePicker() {
    const { ws } = useParams()
    const url = window.location.pathname.split('/')
    if (url[1] === 'ws') {
        url.shift()
    }

    const isSpend =
        url && url[2] ? url[2].includes('spend') || url[2] === 'home' : false
    const currentWorkspace = useAtomValue(workspaceAtom)
    const { value: activeTimeRange, setValue: setActiveTimeRange } =
        useUrlDateRangeState(
            isSpend ? defaultSpendTime(ws || '') : defaultTime(ws || '')
        )

    useEffect(() => {
        if (
            !isSpend &&
            currentWorkspace.current &&
            dayjs(currentWorkspace.current?.createdAt).utc().valueOf() >
                activeTimeRange.start.valueOf()
        ) {
            setActiveTimeRange({
                start: dayjs(currentWorkspace.current?.createdAt).utc(),
                end: activeTimeRange.end,
            })
        }
    }, [currentWorkspace, isSpend])

    const currentValue = () => {
        return {
            start: parseDate(
                activeTimeRange.start.startOf('day').format('YYYY-MM-DD')
            ),
            end: parseDate(
                activeTimeRange.end.endOf('day').format('YYYY-MM-DD')
            ),
        }
    }

    const minValue = () => {
        return parseDate(
            !isSpend && currentWorkspace && currentWorkspace.current
                ? dayjs(currentWorkspace.current?.createdAt)
                      .utc()
                      .format('YYYY-MM-DD')
                : '2022-12-01'
        )
    }
    const maxValue = () => {
        if (isSpend) {
            return today(getLocalTimeZone()).subtract({ days: 2 })
        }
        return today(getLocalTimeZone())
    }

    return (
        <CustomDatePicker
            value={currentValue()}
            onChange={(value) => {
                setActiveTimeRange({
                    start: dayjs.utc(value.start.toString()).startOf('day'),
                    end: dayjs.utc(value.end.toString()).endOf('day'),
                })
            }}
            minValue={minValue()}
            maxValue={maxValue()}
        />
    )
}
