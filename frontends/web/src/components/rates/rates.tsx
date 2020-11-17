/**
 * Copyright 2018 Shift Devices AG
 * Copyright 2020 Shift Crypto AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { h, JSX, RenderableProps } from 'preact';
import { share } from '../../decorators/share';
import { Store } from '../../decorators/store';
import { setConfig } from '../../utils/config';
import { equal } from '../../utils/equal';
import { apiSubscribe } from '../../utils/event';
import { apiGet, apiPost } from '../../utils/request';
import * as style from './rates.css';

export type MainnetCoin = 'BTC' | 'LTC' | 'ETH';

export type TestnetCoin = 'TBTC' | 'TLTC' | 'TETH' | 'RETH';

export type Coin = MainnetCoin | TestnetCoin;

export type Fiat = 'USD' | 'EUR' | 'CHF' | 'GBP' | 'JPY' | 'KRW' | 'CNY' | 'RUB' | 'CAD' | 'AUD' | 'ILS' | 'BTC';

export type Rates = {
    [coin in MainnetCoin]: {
        [fiat in Fiat]: number;
    }
};

export interface SharedProps {
    rates: Rates | undefined | null;
    active: Fiat;
    selected: Fiat[];
}

export const currencies: Fiat[] = ['AUD', 'CAD', 'CHF', 'CNY', 'EUR', 'GBP', 'ILS', 'JPY', 'KRW', 'RUB', 'USD', 'BTC'];

export const store = new Store<SharedProps>({
    rates: undefined,
    active: 'CHF',
    selected: ['USD', 'EUR', 'CHF'],
});

apiGet('config').then((appconf) => {
    if (appconf.frontend && appconf.backend.mainFiat) {
        store.setState({ active: appconf.backend.mainFiat });
    }
    if (appconf.backend && appconf.backend.fiatList) {
        store.setState({ selected: appconf.backend.fiatList });
    }
});

apiGet('rates').then(rates => store.setState({ rates }));

apiSubscribe('rates', ({ object }) => store.setState({ rates: object }));

export function setActiveFiat(fiat: Fiat): void {
    if (!store.state.selected.includes(fiat)) {
        selectFiat(fiat);
    }
    store.setState({ active: fiat });
    setConfig({ backend: { mainFiat: fiat } });
}

export function rotateFiat(): void {
    const index = store.state.selected.indexOf(store.state.active);
    const fiat = store.state.selected[(index + 1) % store.state.selected.length];
    setActiveFiat(fiat);
}

export function selectFiat(fiat: Fiat): void {
    const selected = [...store.state.selected, fiat];
    setConfig({ backend: { fiatList: selected } })
    .then(() => {
        store.setState({ selected });
        // Need to reconfigure currency exchange rates updater
        // which is done during accounts reset.
        apiPost('accounts/reinitialize');
    });
}

export function unselectFiat(fiat: Fiat): void {
    const selected = store.state.selected.filter(item => !equal(item, fiat));
    setConfig({ backend: { fiatList: selected } })
    .then(() => {
        store.setState({ selected });
        // Need to reconfigure currency exchange rates updater
        // which is done during accounts reset.
        apiPost('accounts/reinitialize');
    });
}

export function formatNumber(amount: number): string {
    let formatted = amount.toFixed(2);
    let position = formatted.indexOf('.') - 3;
    while (position > 0) {
        formatted = formatted.slice(0, position) + "'" + formatted.slice(position);
        position = position - 3;
    }
    return formatted;
}

export interface AmountInterface {
    amount: string;
    unit: Coin;
}

interface ProvidedProps {
    amount: AmountInterface;
    tableRow?: boolean;
    unstyled?: boolean;
    skipUnit?: boolean;
    noAction?: boolean;
}

type Props = ProvidedProps & SharedProps;

function Conversion({
    amount,
    tableRow,
    unstyled,
    skipUnit,
    rates,
    active,
    noAction,
    children,
}: RenderableProps<Props>): JSX.Element | null {
    if (!rates) {
        return null;
    }
    const coin = amount.unit;
    let mainnetCoin: MainnetCoin;
    if (coin.length === 4 && coin.startsWith('T') || coin === 'RETH') {
        mainnetCoin = coin.substring(1) as MainnetCoin;
    } else {
        mainnetCoin = coin as MainnetCoin;
    }
    let formattedValue = '';
    if (rates[mainnetCoin]) {
        formattedValue = formatNumber(rates[mainnetCoin][active] * Number(amount.amount));
    }
    if (tableRow) {
        return (
            <tr className={unstyled ? '' : style.fiatRow}>
                <td className={unstyled ? '' : style.availableFiatAmount}>{formattedValue}</td>
                <td className={unstyled ? '' : style.availableFiatUnit} onClick={rotateFiat}>{active}</td>
            </tr>
        );
    }
    return (
        <span className={style.rates}>
            {children}
            {formattedValue}
            {' '}
            {
                !skipUnit && !noAction && (
                    <span className={style.unitAction} onClick={rotateFiat}>{active}</span>
                )
            }
            {
                !skipUnit && noAction && (
                    <span className={style.unit}>{active}</span>
                )
            }
        </span>
    );
}

export const FiatConversion = share<SharedProps, ProvidedProps>(store)(Conversion);
