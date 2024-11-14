import {
    CheckIcon,
    ChevronDownIcon,
    ChevronUpIcon,
} from '@heroicons/react/24/outline'
import { Card, Flex, Text } from '@tremor/react'
import { useState, Fragment, useEffect } from 'react'
import { Listbox, Transition } from '@headlessui/react'
import { capitalizeFirstLetter } from '../../utilities/labelMaker'

function classNames(...classes: (string | null | undefined | boolean)[]) {
    return classes.filter(Boolean).join(' ')
}

interface ISelector {
    values: string[]
    value: string
    title: string
    onValueChange: (value: string) => void
}

export default function Selector({
    values,
    value,
    title,
    onValueChange,
}: ISelector) {
    return (
        <Listbox
            value={values.indexOf(value)}
            onChange={(newValue) => {
                onValueChange(values[newValue])
            }}
        >
            {({ open }) => (
                <Flex
                    flexDirection="col"
                    justifyContent="start"
                    className="w-fit"
                >
                    <div className="relative">
                        <Listbox.Button className="relative w-fit cursor-pointer hover:bg-slate-50 dark:hover:bg-gray-900 hover:ease-in rounded-lg bg-white dark:bg-gray-800 px-4 p-1 text-left text-gray-900 dark:text-gray-50 focus:outline-none sm:text-sm sm:leading-6">
                            <text className="text-xs text-slate-500">
                                {title}
                            </text>
                            <Flex
                                flexDirection="row"
                                alignItems="center"
                                className="gap-1"
                            >
                                <text className="text-md">
                                    {capitalizeFirstLetter(value)}
                                </text>
                                {open ? (
                                    <ChevronUpIcon
                                        className="w-4 ml-1 mt-1"
                                        color="Gray"
                                    />
                                ) : (
                                    <ChevronDownIcon
                                        className="w-4 ml-1 mt-1"
                                        color="Gray"
                                    />
                                )}
                            </Flex>
                        </Listbox.Button>

                        <Transition
                            show={open}
                            as={Fragment}
                            enter="transition ease-out duration-100"
                            enterFrom="opacity-0"
                            enterTo="opacity-100"
                            leave="transition ease-in duration-150"
                            leaveFrom="opacity-100"
                            leaveTo="opacity-0"
                        >
                            <Listbox.Options className="absolute z-10 mt-1 max-h-56 w-fit overflow-auto rounded-md bg-white dark:bg-gray-800 py-1 text-base shadow-lg ring-1 ring-black dark:ring-white ring-opacity-5 focus:outline-none sm:text-sm">
                                {values.map((item, idx) => (
                                    <Listbox.Option
                                        key={item}
                                        className={({ active }) =>
                                            classNames(
                                                active
                                                    ? 'bg-gray-50 dark:bg-gray-900'
                                                    : 'text-gray-900 dark:text-gray-50',
                                                'relative w-full cursor-default select-none py-2 px-4'
                                            )
                                        }
                                        value={idx}
                                    >
                                        {({ selected, active }) => (
                                            <div className="flex flex-row justify-between items-center w-full">
                                                <span
                                                    className={classNames(
                                                        selected
                                                            ? 'font-semibold'
                                                            : 'font-normal',
                                                        ' block truncate text-gray-800 dark:text-gray-100 pr-2'
                                                    )}
                                                >
                                                    {capitalizeFirstLetter(
                                                        item
                                                    )}
                                                </span>

                                                {selected ? (
                                                    <span
                                                        className={classNames(
                                                            active
                                                                ? 'text-gray-900 dark:text-gray-50'
                                                                : 'text-gray-900 dark:text-gray-50',
                                                            'inset-y-0 right-0 flex items-center'
                                                        )}
                                                    >
                                                        <CheckIcon
                                                            className="h-4 w-4"
                                                            aria-hidden="true"
                                                        />
                                                    </span>
                                                ) : null}
                                            </div>
                                        )}
                                    </Listbox.Option>
                                ))}
                            </Listbox.Options>
                        </Transition>
                    </div>
                </Flex>
            )}
        </Listbox>
    )
}
