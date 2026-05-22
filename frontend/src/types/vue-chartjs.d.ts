declare module 'vue-chartjs' {
  import type { DefineComponent } from 'vue'
  import type {
    BubbleDataPoint,
    Chart as ChartJS,
    ChartComponentLike,
    ChartData,
    ChartOptions,
    ChartType,
    DefaultDataPoint,
    Plugin,
    Point,
    UpdateMode,
  } from 'chart.js'

  export interface ChartProps<
    TType extends ChartType = ChartType,
    TData = DefaultDataPoint<TType>,
    TLabel = unknown,
  > {
    type: TType
    data: ChartData<TType, TData, TLabel>
    options?: ChartOptions<TType>
    plugins?: Plugin<TType>[]
    datasetIdKey?: string
    updateMode?: UpdateMode
  }

  export interface ChartComponentRef<
    TType extends ChartType = ChartType,
    TData = DefaultDataPoint<TType>,
    TLabel = unknown,
  > {
    chart: ChartJS<TType, TData, TLabel> | null
  }

  export type ChartComponent = DefineComponent<ChartProps>
  export type TypedChartComponent<
    TType extends ChartType,
    TData = DefaultDataPoint<TType>,
    TLabel = unknown,
  > = DefineComponent<Omit<ChartProps<TType, TData, TLabel>, 'type'>>

  export interface ExtendedDataPoint {
    [key: string]: string | number | null | ExtendedDataPoint
  }

  export function createTypedChart<
    TType extends ChartType = ChartType,
    TData = DefaultDataPoint<TType>,
    TLabel = unknown,
  >(type: TType, registerables: ChartComponentLike): TypedChartComponent<TType, TData, TLabel>

  export const Chart: ChartComponent
  export const Bar: TypedChartComponent<'bar', (number | [number, number] | null)[] | ExtendedDataPoint[], unknown>
  export const Doughnut: TypedChartComponent<'doughnut', number[], unknown>
  export const Line: TypedChartComponent<'line', (number | Point | null)[], unknown>
  export const Pie: TypedChartComponent<'pie', number[], unknown>
  export const PolarArea: TypedChartComponent<'polarArea', number[], unknown>
  export const Radar: TypedChartComponent<'radar', (number | null)[], unknown>
  export const Bubble: TypedChartComponent<'bubble', BubbleDataPoint[], unknown>
  export const Scatter: TypedChartComponent<'scatter', (number | Point | null)[], unknown>
}
