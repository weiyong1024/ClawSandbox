import { h, render } from 'preact';
import { useState, useEffect, useRef, useCallback, useMemo } from 'preact/hooks';
import htm from 'htm';

const html = htm.bind(h);

export { html, render, h, useState, useEffect, useRef, useCallback, useMemo };
