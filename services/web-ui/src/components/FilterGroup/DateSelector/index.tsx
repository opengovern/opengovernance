import { parseDate, today } from '@internationalized/date'
import dayjs from 'dayjs'
import { Checkbox } from 'pretty-checkbox-react'
import { AriaDateRangePickerProps, DateValue } from '@react-aria/datepicker'
import { useDateRangePickerState } from 'react-stately'
import { useEffect, useRef, useState } from 'react'
import { useDateRangePicker } from 'react-aria'
import { Card, Flex, Text, Title, NumberInput } from '@tremor/react'
import { ClockIcon } from '@heroicons/react/24/outline'
import DateConditionSelector, {
    DateSelectorOptions,
} from '../ConditionSelector/DateConditionSelector'
import { DateRange } from '../../../utilities/urlstate'
import { RangeCalendar } from '../../Layout/Header/DatePicker/Calendar/RangePicker/RangeCalendar'

export interface IDateSelector {
    title: string
    value: DateRange
    onValueChanged: (value: DateRange) => void
    supportedConditions: DateSelectorOptions[]
    selectedCondition: DateSelectorOptions
    onConditionChange: (condition: DateSelectorOptions) => void
}

export interface IDate {
    start: dayjs.Dayjs
    end: dayjs.Dayjs
}

function CustomDatePicker(props: AriaDateRangePickerProps<DateValue>) {
    const state = useDateRangePickerState(props)
    const ref = useRef(null)

    const { calendarProps } = useDateRangePicker(props, state, ref)

    return <RangeCalendar {...calendarProps} />
}

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

export default function DateSelector({
    title,
    value,
    supportedConditions,
    selectedCondition,
    onValueChanged,
    onConditionChange,
}: IDateSelector) {
    const startH = value.start.hour()
    const startM = value.start.minute()
    const endH = value.end.hour()
    const endM = value.end.minute()

    const [checked, setChecked] = useState(
        startH !== 0 || startM !== 0 || endH !== 23 || endM !== 59
    )

    useEffect(() => {
        if (!checked) {
            onValueChanged({
                start: dayjs(value.start).startOf('day'),
                end: dayjs(value.end).endOf('day'),
            })
        }
    }, [checked])

    const minValue = () => {
        return parseDate('2022-12-01')
    }
    const maxValue = () => {
        return today('UTC')
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
        <Card className="mt-2 py-4 px-6 w-fit rounded-xl">
            <Flex>
                <Flex
                    justifyContent="start"
                    alignItems="baseline"
                    className="gap-2"
                >
                    <Text>{title}</Text>
                    <DateConditionSelector
                        supportedConditions={supportedConditions}
                        selectedCondition={selectedCondition}
                        onConditionChange={(i) => onConditionChange(i)}
                    />
                </Flex>
            </Flex>

            <Flex
                flexDirection="col"
                alignItems="start"
                className="gap-2 my-4 overflow-auto"
            >
                {selectedCondition === 'isRelative' ? (
                    <>
                        <Flex
                            onClick={() => onValueChanged(last7Days())}
                            className="mt-4 px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                Last 7 days
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    last7Days().start,
                                    last7Days().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => onValueChanged(last30Days())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                Last 30 days
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    last30Days().start,
                                    last30Days().end
                                )}
                            </Text>
                        </Flex>
                        <Title className="mt-3">Calender months</Title>
                        <Flex
                            onClick={() => onValueChanged(thisMonth())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                This month
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    thisMonth().start,
                                    thisMonth().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => onValueChanged(lastMonth())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                Last month
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    lastMonth().start,
                                    lastMonth().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => onValueChanged(thisQuarter())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                This quarter
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    thisQuarter().start,
                                    thisQuarter().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => onValueChanged(lastQuarter())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                Last quarter
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    lastQuarter().start,
                                    lastQuarter().end
                                )}
                            </Text>
                        </Flex>
                        <Flex
                            onClick={() => onValueChanged(thisYear())}
                            className="px-4 space-x-4 py-2 cursor-pointer rounded-md hover:bg-openg-50 dark:hover:bg-openg-700"
                        >
                            <Text className="text-gray-800 whitespace-nowrap">
                                This year
                            </Text>
                            <Text className="whitespace-nowrap">
                                {renderDateText(
                                    thisYear().start,
                                    thisYear().end
                                )}
                            </Text>
                        </Flex>
                    </>
                ) : (
                    <>
                        <CustomDatePicker
                            value={{
                                start: parseDate(
                                    value.start.format('YYYY-MM-DD')
                                ),
                                end: parseDate(value.end.format('YYYY-MM-DD')),
                            }}
                            onChange={(v) => {
                                onValueChanged({
                                    start: dayjs(v.start.toString()),
                                    end: dayjs(v.end.toString()).endOf('day'),
                                })
                            }}
                            minValue={minValue()}
                            maxValue={maxValue()}
                        />
                        <Checkbox
                            checked={checked}
                            onChange={(e) => {
                                setChecked(e.target.checked)
                            }}
                            className="my-3"
                        >
                            <Flex className="gap-1">
                                <ClockIcon className="w-4" />
                                <Text>Time</Text>
                            </Flex>
                        </Checkbox>
                        {checked && (
                            <Flex flexDirection="col" className="gap-2 w-full">
                                <Flex className="gap-3" justifyContent="start">
                                    <Text className="w-16 whitespace-nowrap">
                                        Start time
                                    </Text>
                                    <Flex className="gap-2 w-20">
                                        <NumberInput
                                            placeholder="HH"
                                            value={startH}
                                            min={0}
                                            max={24}
                                            onValueChange={(x) => {
                                                onValueChanged({
                                                    ...value,
                                                    start: value.start.set(
                                                        'hour',
                                                        Number(x)
                                                    ),
                                                })
                                            }}
                                        />
                                        <Title>:</Title>
                                        <NumberInput
                                            placeholder="HH"
                                            value={startM}
                                            min={0}
                                            max={60}
                                            onValueChange={(x) => {
                                                onValueChanged({
                                                    ...value,
                                                    start: value.start.set(
                                                        'minute',
                                                        Number(x)
                                                    ),
                                                })
                                            }}
                                        />
                                    </Flex>
                                </Flex>
                                <Flex className="gap-3">
                                    <Text className="w-16 whitespace-nowrap">
                                        End time
                                    </Text>
                                    <Flex className="w-full gap-2">
                                        <NumberInput
                                            placeholder="HH"
                                            value={endH}
                                            min={0}
                                            max={24}
                                            onValueChange={(x) => {
                                                onValueChanged({
                                                    ...value,
                                                    start: value.end.set(
                                                        'hour',
                                                        Number(x)
                                                    ),
                                                })
                                            }}
                                        />
                                        <Title>:</Title>
                                        <NumberInput
                                            placeholder="HH"
                                            value={endM}
                                            min={0}
                                            max={60}
                                            onValueChange={(x) => {
                                                onValueChanged({
                                                    ...value,
                                                    start: value.end.set(
                                                        'minute',
                                                        Number(x)
                                                    ),
                                                })
                                            }}
                                        />
                                    </Flex>
                                </Flex>
                            </Flex>
                        )}
                    </>
                )}
            </Flex>
        </Card>
    )
}
