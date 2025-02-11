/**
 * Copyright 2022-2025 Shift Crypto AG
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

import { useTranslation } from 'react-i18next';
import type { IAccount } from '@/api/account';
import type { TExchangeName } from '@/api/exchanges';
import { i18n } from '@/i18n/i18n';
import { A } from '@/components/anchor/anchor';
import { isBitcoinOnly } from '@/routes/account/utils';
import style from './infocontent.module.css';

export const getBTCDirectOTCLink = () => {
  switch (i18n.resolvedLanguage) {
  case 'de':
    return 'https://btcdirect.eu/de-at/private-trading-contact?BitBox';
  case 'nl':
    return 'https://btcdirect.eu/nl-nl/private-trading-contact?BitBox';
  case 'es':
    return 'https://btcdirect.eu/es-es/private-trading-contactanos?BitBox';
  case 'fr':
    return 'https://btcdirect.eu/fr-fr/private-trading-contact?BitBox';
  default:
    return 'https://btcdirect.eu/en-eu/private-trading-contact?BitBox';
  }
};

export const getBTCDirectLink = () => {
  switch (i18n.resolvedLanguage) {
  case 'de':
    return 'https://btcdirect.eu/de-at/uber-uns?BitBox';
  case 'nl':
    return 'https://btcdirect.eu/nl-nl/over-ons?BitBox';
  case 'es':
    return 'https://btcdirect.eu/es-es/sobre-nosotros?BitBox';
  case 'fr':
    return 'https://btcdirect.eu/fr-fr/a-propos-de-nous?BitBox';
  default:
    return 'https://btcdirect.eu/en-eu/about-btc-direct?BitBox';
  }
};

type TMoonPayInfoProps = {
  cardFee?: number;
  bankTransferFee?: number;
};

const MoonPayInfo = ({ cardFee, bankTransferFee }: TMoonPayInfoProps) => {
  const { t } = useTranslation();
  const formattedCardFee = cardFee && cardFee * 100;
  const formattedBankTransferFee = bankTransferFee && bankTransferFee * 100;
  return (
    <div className={style.container}>
      <p>{t('buy.exchange.infoContent.moonpay.supportedCurrencies')}</p>
      <br />
      <p><A href="https://support.moonpay.com/hc/en-gb/articles/360011931457-Which-fiat-currencies-are-supported-">{t('buy.exchange.infoContent.moonpay.fullCurrenciesList')}</A></p>
      <br />
      <p><b>{t('buy.exchange.infoContent.moonpay.payment.title')}</b></p>
      <br />
      <p>{t('buy.exchange.infoContent.moonpay.payment.creditDebitCard')}</p>
      <ul>
        <li>{t('buy.exchange.infoContent.moonpay.payment.creditDebitCardDetails.cards')}</li>
      </ul>
      <p>{t('buy.exchange.infoContent.moonpay.payment.bankTransfer')}</p>
      <ul>
        <li>{t('buy.exchange.infoContent.moonpay.payment.bankTransferDetails.sepa')}</li>
        <li>{t('buy.exchange.infoContent.moonpay.payment.bankTransferDetails.uk')}</li>
        <li>{t('buy.exchange.infoContent.moonpay.payment.bankTransferDetails.pix')}</li>
      </ul>
      <br/>
      <p><i>{t('buy.exchange.infoContent.moonpay.payment.asteriskText')}</i></p>
      <br />
      <p><A href="https://support.moonpay.com/hc/en-gb/articles/4406210084113-What-payment-methods-do-you-support-">{t('buy.exchange.infoContent.moonpay.payment.learnMore')}</A></p>
      <br />
      <p><b>{t('buy.exchange.infoContent.moonpay.fees.title')}</b></p>
      <ul>
        <li>{t('buy.exchange.infoContent.moonpay.fees.creditDebitCard', { fee: formattedCardFee })}</li>
        <li>{t('buy.exchange.infoContent.moonpay.fees.bankTransfer', { fee: formattedBankTransferFee })}</li>
      </ul>
      <br />
      <p><A href="https://www.moonpay.com/">{t('buy.exchange.infoContent.moonpay.fees.learnMore')}</A></p>
    </div>
  );
};

type TPocketInfoProps = {
  bankTransferFee?: number
};

const PocketInfo = ({ bankTransferFee }: TPocketInfoProps) => {
  const { t } = useTranslation();
  const fee = bankTransferFee && bankTransferFee * 100;
  return (
    <div className={style.container}>
      <p>{t('buy.exchange.infoContent.pocket.supportedCurrencies')}</p>
      <br />
      <p><b>{t('buy.exchange.infoContent.pocket.payment.title')}</b></p>
      <br />
      <p>{t('buy.exchange.infoContent.pocket.payment.bankTransfer')}</p>
      <ul>
        <li>{t('buy.exchange.infoContent.pocket.payment.bankTransferDetails.sepa')}</li>
        <li>{t('buy.exchange.infoContent.pocket.payment.bankTransferDetails.uk')}</li>
        <li>{t('buy.exchange.infoContent.pocket.payment.bankTransferDetails.sic')}</li>
      </ul>
      <br/>
      <p>
        <A href="https://pocketbitcoin.com/faq/how-do-I-set-up-my-standing-order">
          {t('buy.exchange.infoContent.pocket.payment.bankTransferReccuring')}
        </A>
      </p>
      <br />
      <p><b>{t('buy.exchange.infoContent.pocket.verification.title')}</b></p>
      <br />
      <p>{t('buy.exchange.infoContent.pocket.verification.info')}</p>
      <br />
      <p>
        <A href="https://pocketbitcoin.com/faq/are-there-any-limits-with-pocket">
          {t('buy.exchange.infoContent.pocket.verification.link')}
        </A>
      </p>
      <br />
      <p><b>{t('buy.exchange.infoContent.pocket.fees.title')}</b></p>
      <ul>
        <li>{t('buy.exchange.infoContent.pocket.fees.info', { fee })}</li>
      </ul>
      <br />
      <p>
        <A href="https://pocketbitcoin.com/">
          {t('buy.exchange.infoContent.pocket.learnMore')}
        </A>
      </p>
      <br />
      <p><b>{t('buy.exchange.infoContent.pocket.sell.title')}</b></p>
      <br />
      <p>{t('exchange.pocket.terms.fees.extraNote')}</p>
    </div>
  );
};

type TBTCDirectOTCInfoProps = {
  accounts?: IAccount[];
};

const BTCDirectOTCInfo = ({ accounts }: TBTCDirectOTCInfoProps) => {
  const { t } = useTranslation();
  const hasOnlyBTCAccounts = accounts?.every(({ coinCode }) => isBitcoinOnly(coinCode));
  return (
    <div className={style.container}>
      <p>
        {t('buy.exchange.infoContent.btcdirect.infobox.intro', {
          context: hasOnlyBTCAccounts ? 'btconly' : 'multi'
        })}
      </p>
      <br />
      <p>{t('buy.exchange.infoContent.btcdirect.infobox.manager')}</p>
      <br />
      <ul>
        <li>{t('buy.exchange.infoContent.btcdirect.infobox.listItem1')}</li>
        <li>{t('buy.exchange.infoContent.btcdirect.infobox.listItem2')}</li>
        <li>{t('buy.exchange.infoContent.btcdirect.infobox.listItem3')}</li>
        <li>{t('buy.exchange.infoContent.btcdirect.infobox.listItem4')}</li>
      </ul>
      <br />
      <p>{t('buy.exchange.infoContent.btcdirect.infobox.kyc')}</p>
      <br />
      <p>
        <A href={getBTCDirectOTCLink()}>
          {t('buy.exchange.infoContent.btcdirect.infobox.learnmore')}
        </A>
      </p>
    </div>
  );
};

type TBTCDirectInfoProps = {
  cardFee?: number;
  bankTransferFee?: number;
  sofortFee?: number;
  bancontactFee?: number;
};

const BTCDirectInfo = ({
  cardFee,
  bankTransferFee,
  sofortFee,
  bancontactFee,
}: TBTCDirectInfoProps) => {
  const { t } = useTranslation();
  const formattedCardFee = cardFee && cardFee * 100;
  const formattedBankTransferFee = bankTransferFee && bankTransferFee * 100;
  const formattedSofortFee = sofortFee && sofortFee * 100;
  const formattedBancontactFee = bancontactFee && bancontactFee * 100;
  return (
    <div className={style.container}>
      <p>{t('buy.exchange.infoContent.btcdirectWidget.infobox.intro')}</p>
      <br />
      <p><b>{t('buy.exchange.infoContent.btcdirectWidget.infobox.payment.title')}</b></p>
      <br />
      <ul>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.payment.bankTransfer')}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.payment.creditDebitCard')}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.payment.sofort')}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.payment.bancontact')}</li>
      </ul>
      <br />
      <p><b>{t('buy.exchange.infoContent.btcdirectWidget.infobox.feesBuying.title')}</b></p>
      <br />
      <ul>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.feesBuying.bankTransfer', {
          fee: formattedBankTransferFee
        })}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.feesBuying.creditDebitCard', {
          fee: formattedCardFee
        })}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.feesBuying.sofort', {
          fee: formattedSofortFee
        })}</li>
        <li>{t('buy.exchange.infoContent.btcdirectWidget.infobox.feesBuying.bancontact', {
          fee: formattedBancontactFee
        })}</li>
      </ul>
      <br />
      <p>
        <A href={getBTCDirectLink()}>
          {t('buy.exchange.infoContent.btcdirectWidget.infobox.learnmore')}
        </A>
      </p>
    </div>
  );
};

const RegionInfo = () => {
  const { t } = useTranslation();
  return (
    <div>
      <p>{t('buy.exchange.infoContent.region.title')}</p>
    </div>
  );
};

type Info = TExchangeName | 'region';

export type TInfoContentProps = {
  accounts?: IAccount[];
  bancontactFee?: number;
  bankTransferFee?: number;
  cardFee?: number;
  info: Info;
  sofortFee?: number;
};

export const InfoContent = ({
  accounts,
  bancontactFee,
  bankTransferFee,
  cardFee,
  info,
  sofortFee,
}: TInfoContentProps) => {
  switch (info) {
  case 'moonpay':
    return (
      <MoonPayInfo
        cardFee={cardFee}
        bankTransferFee={bankTransferFee}
      />
    );
  case 'pocket':
    return (
      <PocketInfo bankTransferFee={bankTransferFee} />
    );
  case 'btcdirect':
    return (
      <BTCDirectInfo
        cardFee={cardFee}
        bankTransferFee={bankTransferFee}
        bancontactFee={bancontactFee}
        sofortFee={sofortFee}
      />
    );
  case 'btcdirect-otc':
    return (
      <BTCDirectOTCInfo accounts={accounts} />
    );
  case 'region':
    return (
      <RegionInfo />
    );
  }
  return null;
};
