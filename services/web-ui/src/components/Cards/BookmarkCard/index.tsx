
import { ChevronRightIcon } from "@heroicons/react/24/outline";
import {
  ComponentType,
  FunctionComponent,
  ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

interface CardProps {
    title: string
    logos?: string[]
    tag?: string
    onClick?: () => void
    option?: string
    description?: string
}

const UseCaseCard: FunctionComponent<CardProps> = ({
  title,
  
  tag,
  onClick,
  option
  ,description,
  logos
}) => {
  const truncate = (text: string | undefined, number: number) => {
    if (text) {
      return text.length > number ? text.substring(0, number) + "..." : text;
    }
  };

  return (
      <>
          <div
              onClick={() => {
                  onClick?.()
              }}
              className="card cursor-pointer rounded-lg border shadow-2xl dark:border-none dark:bg-white h-full flex flex-col justify-between  w-full gap-4 "
          >
              <div className="flex flex-row justify-between rounded-xl  items-center px-4 py-2">
                  <div className="flex flex-row gap-2">
                     {logos?.map((logo) => {
                        return (
                            <div className=" bg-gray-300 dark:bg-slate-400 rounded p-2">
                                <img
                                    src={logo}
                                    className=" h-5 w-5"
                                    onError={(e) => {
                                        e.currentTarget.onerror = null
                                        e.currentTarget.src =
                                            'https://raw.githubusercontent.com/opengovern/website/main/connectors/icons/default.svg'
                                    }}
                                />
                            </div>
                        )
                     })}
                  </div>
                  <div>
                      {/* <span className="rounded-3xl text-black dark:text-white bg-gray-300 dark:bg-slate-400 px-3 py-1 text-center">
              {tag}
            </span> */}
                  </div>
              </div>
              <div className=" text-start flex flex-col gap-1 text-black text-wrap px-4  ">
                  <span className=" text-xl font-bold">{title}</span>
                  <span className="text-sm text-gray-500">
                    {description}
                  </span>
              </div>

              <div className="flex flex-row justify-center w-full bg-openg-950 dark:bg-blue-900 rounded-b-lg px-4 py-2 items-center">
                  {/* <span className="dark:text-white">google sheet + some text </span> */}
                  <div className="flex w-full text-white flex-row justify-center items-center gap-2">
                      <span>Run it</span>
                      <ChevronRightIcon className="w-5" color="white" />
                  </div>
              </div>
          </div>
      </>
  )
};

export default UseCaseCard;
